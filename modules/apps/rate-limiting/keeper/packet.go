package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

type RateLimitedPacketInfo struct {
	ChannelID string
	Denom     string
	Amount    sdkmath.Int
	Sender    string
	Receiver  string
}

// CheckAcknowledementSucceeded unmarshals IBC Acknowledgements, and determines
// whether the tx was successful
func (k *Keeper) CheckAcknowledementSucceeded(ctx sdk.Context, ack []byte) (bool, error) {
	// Check if the ack is the IBC v2 universal error acknowledgement
	if bytes.Equal(ack, channeltypesv2.ErrorAcknowledgement[:]) {
		return false, nil
	}

	// Unmarshal the raw ack response
	var acknowledgement channeltypes.Acknowledgement
	if err := transfertypes.ModuleCdc.UnmarshalJSON(ack, &acknowledgement); err != nil {
		return false, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %s", err.Error())
	}

	// The ack can come back as either AcknowledgementResult or AcknowledgementError
	// If it comes back as AcknowledgementResult, the messages are encoded differently depending on the SDK version
	switch response := acknowledgement.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		if len(response.Result) == 0 {
			return false, errorsmod.Wrapf(channeltypes.ErrInvalidAcknowledgement, "acknowledgement result cannot be empty")
		}
		return true, nil

	case *channeltypes.Acknowledgement_Error:
		k.Logger(ctx).Error(fmt.Sprintf("acknowledgement error: %s", response.Error))
		return false, nil

	default:
		return false, errorsmod.Wrapf(channeltypes.ErrInvalidAcknowledgement, "unsupported acknowledgement response field type %T", response)
	}
}

// ParseDenomFromSendPacket parses the denom from the Send Packet.
// The denom that the rate limiter will use for a SEND packet depends on whether
// it was a NATIVE token (e.g. ustrd, stuatom, etc.) or NON-NATIVE token (e.g. ibc/...)...
//
// We can identify if the token is native or not by parsing the trace denom from the packet
// If the token is NATIVE, it will not have a prefix (e.g. ustrd),
// and if it is NON-NATIVE, it will have a prefix (e.g. transfer/channel-2/uosmo)
//
// For NATIVE denoms, return as is (e.g. ustrd)
// For NON-NATIVE denoms, take the ibc hash (e.g. hash "transfer/channel-2/usoms" into "ibc/...")
func ParseDenomFromSendPacket(packet transfertypes.FungibleTokenPacketData) string {
	// Check if the denom is already an IBC denom (starts with "ibc/")
	if strings.HasPrefix(packet.Denom, "ibc/") {
		return packet.Denom
	}

	// Determine the denom by looking at the denom trace path
	denom := transfertypes.ExtractDenomFromPath(packet.Denom)
	return denom.IBCDenom()
}

// ParseDenomFromRecvPacket parses the denom from the Recv Packet that will be used by the rate limit module.
// The denom that the rate limiter will use for a RECEIVE packet depends on whether it was a source or sink.
//
//	Sink:   The token moves forward, to a chain different than its previous hop
//	        The new port and channel are APPENDED to the denom trace.
//	        (e.g. A -> B, B is a sink) (e.g. A -> B -> C, C is a sink)
//
//	Source: The token moves backwards (i.e. revisits the last chain it was sent from)
//	        The port and channel are REMOVED from the denom trace - undoing the last hop.
//	        (e.g. A -> B -> A, A is a source) (e.g. A -> B -> C -> B, B is a source)
//
//	If the chain is acting as a SINK: We add on the port and channel and hash it
//	  Ex1: uosmo sent from Osmosis to Stride
//	       Packet Denom:   uosmo
//	        -> Add Prefix: transfer/channel-X/uosmo
//	        -> Hash:       ibc/...
//
//	  Ex2: ujuno sent from Osmosis to Stride
//	       PacketDenom:    transfer/channel-Y/ujuno  (channel-Y is the Juno <> Osmosis channel)
//	        -> Add Prefix: transfer/channel-X/transfer/channel-Y/ujuno
//	        -> Hash:       ibc/...
//
//	If the chain is acting as a SOURCE: First, remove the prefix. Then if there is still a denom trace, hash it
//	  Ex1: ustrd sent back to Stride from Osmosis
//	       Packet Denom:      transfer/channel-X/ustrd
//	        -> Remove Prefix: ustrd
//	        -> Leave as is:   ustrd
//
//	  Ex2: juno was sent to Stride, then to Osmosis, then back to Stride
//	       Packet Denom:      transfer/channel-X/transfer/channel-Z/ujuno
//	        -> Remove Prefix: transfer/channel-Z/ujuno
//	        -> Hash:          ibc/...
func ParseDenomFromRecvPacket(packet channeltypes.Packet, packetData transfertypes.FungibleTokenPacketData) string {
	sourcePort := packet.SourcePort
	sourceChannel := packet.SourceChannel

	// To determine the denom, first check whether Stride is acting as source
	// Build the source prefix and check if the denom starts with it
	hop := transfertypes.NewHop(sourcePort, sourceChannel)
	sourcePrefix := hop.String() + "/"

	if strings.HasPrefix(packetData.Denom, sourcePrefix) {
		// Remove the source prefix (e.g. transfer/channel-X/transfer/channel-Z/ujuno -> transfer/channel-Z/ujuno)
		unprefixedDenom := packetData.Denom[len(sourcePrefix):]

		// Native assets will have an empty trace path and can be returned as is
		denom := transfertypes.ExtractDenomFromPath(unprefixedDenom)
		return denom.IBCDenom()
	}
	// Prefix the destination channel - this will contain the trailing slash (e.g. transfer/channel-X/)
	destinationPrefix := transfertypes.NewHop(packet.GetDestPort(), packet.GetDestChannel())
	prefixedDenom := destinationPrefix.String() + "/" + packetData.Denom

	// Hash the denom trace
	denom := transfertypes.ExtractDenomFromPath(prefixedDenom)
	return denom.IBCDenom()
}

// ParsePacketInfo parses the sender and channelId and denom for the corresponding RateLimit object, and
// the sender/receiver/transfer amount
//
// The channelID should always be used as the key for the RateLimit object (not the counterparty channelID)
// For a SEND packet, the channelID is the SOURCE channel
// For a RECEIVE packet, the channelID is the DESTINATION channel
//
// The Source and Destination are defined from the perspective of a packet recipient.
func ParsePacketInfo(packet channeltypes.Packet, direction types.PacketDirection) (RateLimitedPacketInfo, error) {
	var packetData transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &packetData); err != nil {
		return RateLimitedPacketInfo{}, err
	}

	var channelID, denom string
	if direction == types.PACKET_SEND {
		channelID = packet.GetSourceChannel()
		denom = ParseDenomFromSendPacket(packetData)
	} else {
		channelID = packet.GetDestChannel()
		denom = ParseDenomFromRecvPacket(packet, packetData)
	}

	amount, ok := sdkmath.NewIntFromString(packetData.Amount)
	if !ok {
		return RateLimitedPacketInfo{},
			errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "Unable to cast packet amount '%s' to sdkmath.Int", packetData.Amount)
	}

	packetInfo := RateLimitedPacketInfo{
		ChannelID: channelID,
		Denom:     denom,
		Amount:    amount,
		Sender:    packetData.Sender,
		Receiver:  packetData.Receiver,
	}

	return packetInfo, nil
}

// Middleware implementation for SendPacket with rate limiting
// Checks whether the rate limit has been exceeded - and if it hasn't, sends the packet
func (k *Keeper) SendRateLimitedPacket(ctx sdk.Context, sourcePort, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) error {
	seq, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return errorsmod.Wrapf(channeltypes.ErrSequenceSendNotFound, "source port: %s, source channel: %s", sourcePort, sourceChannel)
	}

	packet := channeltypes.Packet{
		Sequence:         seq,
		SourcePort:       sourcePort,
		SourceChannel:    sourceChannel,
		TimeoutHeight:    timeoutHeight,
		TimeoutTimestamp: timeoutTimestamp,
		Data:             data,
	}

	packetInfo, err := ParsePacketInfo(packet, types.PACKET_SEND)
	if err != nil {
		return err
	}

	// Check if the packet would exceed the outflow rate limit
	updatedFlow, err := k.CheckRateLimitAndUpdateFlow(ctx, types.PACKET_SEND, packetInfo)
	if err != nil {
		return err
	}

	// Store the sequence number of the packet so that if the transfer fails,
	// we can identify if it was sent during this quota and can revert the outflow
	if updatedFlow {
		k.SetPendingSendPacket(ctx, packetInfo.ChannelID, packet.Sequence)
	}

	return nil
}

// Middleware implementation for RecvPacket with rate limiting
// Checks whether the rate limit has been exceeded - and if it hasn't, allows the packet
func (k *Keeper) ReceiveRateLimitedPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	packetInfo, err := ParsePacketInfo(packet, types.PACKET_RECV)
	if err != nil {
		// If the packet data is unparseable, we can't apply rate limiting.
		// Log the error and allow the packet to proceed to the underlying app
		// which is responsible for handling invalid packet data.
		k.Logger(ctx).Error("Unable to parse packet data for rate limiting", "error", err)
		return nil // Returning nil allows the packet to continue down the stack
	}

	// If parsing was successful, check the rate limit
	_, err = k.CheckRateLimitAndUpdateFlow(ctx, types.PACKET_RECV, packetInfo)
	// If CheckRateLimitAndUpdateFlow returns an error (e.g., quota exceeded), return it to generate an error ack.
	return err
}

// AcknowledgeRateLimitedPacket implements for OnAckPacket for porttypes.Middleware.
// If the packet failed, we should decrement the Outflow.
func (k *Keeper) AcknowledgeRateLimitedPacket(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte) error {
	ackSuccess, err := k.CheckAcknowledementSucceeded(ctx, acknowledgement)
	if err != nil {
		return err
	}

	// Parse the denom, channelId, and amount from the packet
	packetInfo, err := ParsePacketInfo(packet, types.PACKET_SEND)
	if err != nil {
		return err
	}

	// If the ack was successful, remove the pending packet
	if ackSuccess {
		k.RemovePendingSendPacket(ctx, packetInfo.ChannelID, packet.Sequence)
		return nil
	}

	// If the ack failed, undo the change to the rate limit Outflow
	return k.UndoSendPacket(ctx, packetInfo.ChannelID, packet.Sequence, packetInfo.Denom, packetInfo.Amount)
}

// Middleware implementation for OnAckPacket with rate limiting
// The Outflow should be decremented from the failed packet
func (k *Keeper) TimeoutRateLimitedPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	packetInfo, err := ParsePacketInfo(packet, types.PACKET_SEND)
	if err != nil {
		return err
	}

	return k.UndoSendPacket(ctx, packetInfo.ChannelID, packet.Sequence, packetInfo.Denom, packetInfo.Amount)
}
