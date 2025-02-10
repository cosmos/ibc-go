package v2

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/events"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/telemetry"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var _ api.IBCModule = (*IBCModule)(nil)

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper) *IBCModule {
	return &IBCModule{
		keeper: k,
	}
}

type IBCModule struct {
	keeper keeper.Keeper
}

func (im *IBCModule) OnSendPacket(goCtx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
	// Enforce that the source and destination portIDs are the same and equal to the transfer portID
	// This is necessary for IBC Eureka since the portIDs (and thus the application-application connection) is not prenegotiated
	// by the channel handshake
	// This restriction can be removed in a future where the trace hop on receive commits to **both** the source and destination portIDs
	// rather than just the destination port
	if payload.SourcePort != types.PortID || payload.DestinationPort != types.PortID {
		return errorsmod.Wrapf(channeltypesv2.ErrInvalidPacket, "payload port ID is invalid: expected %s, got sourcePort: %s destPort: %s", types.PortID, payload.SourcePort, payload.DestinationPort)
	}
	data, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	sender, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return err
	}

	if !signer.Equals(sender) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "sender %s is different from signer %s", sender, signer)
	}

	// Enforce that the base denom does not contain any slashes
	// Since IBC v2 packets will no longer have channel identifiers, we cannot rely
	// on the channel format to easily divide the trace from the base denomination in ICS20 v1 packets
	// The simplest way to prevent any potential issues from arising is to simply disallow any slashes in the base denomination
	// This prevents such denominations from being sent with IBCV v2 packets, however we can still support them in IBC v1 packets
	// If we enforce that IBC v2 packets are sent with ICS20 v2 and above versions that separate the trace from the base denomination
	// in the packet data, then we can remove this restriction.
	for _, token := range data.Tokens {
		if strings.Contains(token.Denom.Base, "/") {
			return errorsmod.Wrapf(types.ErrInvalidDenomForTransfer, "base denomination %s cannot contain slashes for IBC v2 packet", token.Denom.Base)
		}
	}

	if err := im.keeper.SendTransfer(sdk.UnwrapSDKContext(goCtx), payload.SourcePort, sourceChannel, data.Tokens, signer); err != nil {
		return err
	}

	events.EmitTransferEvent(goCtx, sender.String(), data.Receiver, data.Tokens, data.Memo, data.Forwarding.Hops)

	telemetry.ReportTransfer(payload.SourcePort, sourceChannel, payload.DestinationPort, destinationChannel, data.Tokens)

	return nil
}

func (im *IBCModule) OnRecvPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	// Enforce that the source and destination portIDs are the same and equal to the transfer portID
	// This is necessary for IBC Eureka since the portIDs (and thus the application-application connection) is not prenegotiated
	// by the channel handshake
	// This restriction can be removed in a future where the trace hop on receive commits to **both** the source and destination portIDs
	// rather than just the destination port
	if payload.SourcePort != types.PortID || payload.DestinationPort != types.PortID {
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}
	var (
		ackErr error
		data   types.FungibleTokenPacketDataV2
	)

	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	recvResult := channeltypesv2.RecvPacketResult{
		Status:          channeltypesv2.PacketStatus_Success,
		Acknowledgement: ack.Acknowledgement(),
	}
	// we are explicitly wrapping this emit event call in an anonymous function so that
	// the packet data is evaluated after it has been assigned a value.
	defer func() {
		events.EmitOnRecvPacketEvent(ctx, data, ack, ackErr)
	}()

	data, ackErr = types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if ackErr != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), sequence))
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	if _, ackErr = im.keeper.OnRecvPacket(
		ctx,
		data,
		payload.SourcePort,
		sourceChannel,
		payload.DestinationPort,
		destinationChannel,
	); ackErr != nil {
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), sequence))
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
	}

	im.keeper.Logger(ctx).Info("successfully handled ICS-20 packet", "sequence", sequence)

	telemetry.ReportOnRecvPacket(payload.SourcePort, sourceChannel, payload.DestinationPort, destinationChannel, data.Tokens)

	if data.HasForwarding() {
		// we are now sending from the forward escrow address to the final receiver address.
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Failure,
		}
		// TODO: handle forwarding
		// TODO: inside this version of the function, we should fetch the packet that was stored in IBC core in order to set it for forwarding.
		//	if err := k.forwardPacket(ctx, data, packet, receivedCoins); err != nil {
		//		return err
		//	}

		// NOTE: acknowledgement will be written asynchronously
		// return types.RecvPacketResult{
		// 	Status: types.PacketStatus_Async,
		// }
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return recvResult
}

func (im *IBCModule) OnTimeoutPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	data, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	// refund tokens
	if err := im.keeper.OnTimeoutPacket(ctx, payload.SourcePort, sourceChannel, data); err != nil {
		return err
	}

	// TODO: handle forwarding

	events.EmitOnTimeoutEvent(ctx, data)

	return nil
}

func (im *IBCModule) OnAcknowledgementPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	var ack channeltypes.Acknowledgement
	// construct an error acknowledgement if the acknowledgement bytes are the sentinel error acknowledgement so we can use the shared transfer logic
	if bytes.Equal(acknowledgement, channeltypesv2.ErrorAcknowledgement[:]) {
		// the specific error does not matter
		ack = channeltypes.NewErrorAcknowledgement(types.ErrReceiveFailed)
	} else {
		if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
			return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
		}
		if !ack.Success() {
			return errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot pass in a custom error acknowledgement with IBC v2")
		}
	}

	data, err := types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, payload.SourcePort, sourceChannel, data, ack); err != nil {
		return err
	}

	// TODO: handle forwarding

	events.EmitOnAcknowledgementPacketEvent(ctx, data, ack)

	return nil
}

// UnmarshalPacketData unmarshals the ICS20 packet data based on the version and encoding
// it implements the PacketDataUnmarshaler interface
func (*IBCModule) UnmarshalPacketData(payload channeltypesv2.Payload) (interface{}, error) {
	return types.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
}
