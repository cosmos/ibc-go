package keeper

import (
	"strings"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	coretypes "github.com/cosmos/ibc-go/v3/modules/core/types"
)

// SendTransfer handles nft-transfer sending logic.
// A sending chain may be acting as a source or sink zone.
//
// when a chain is sending tokens across a port and channel which are
// not equal to the last prefixed port and channel pair, it is acting as a source zone.
// when tokens are sent from a source zone, the destination port and
// channel will be prefixed onto the classId (once the tokens are received)
// adding another hop to the tokens record.
//
// when a chain is sending tokens across a port and channel which are
// equal to the last prefixed port and channel pair, it is acting as a sink zone.
// when tokens are sent from a sink zone, the last prefixed port and channel
// pair on the classId is removed (once the tokens are received), undoing the last hop in the tokens record.
//
// For example, assume these steps of transfer occur:
// A -> B -> C -> A -> C -> B -> A
//
//|                    sender  chain                      |                       receiver     chain              |
//| :-----: | -------------------------: | :------------: | :------------: | -------------------------: | :-----: |
//|  chain  |                    classID | (port,channel) | (port,channel) |                    classID |  chain  |
//|    A    |                   nftClass |    (p1,c1)     |    (p2,c2)     |             p2/c2/nftClass |    B    |
//|    B    |             p2/c2/nftClass |    (p3,c3)     |    (p4,c4)     |       p4/c4/p2/c2/nftClass |    C    |
//|    C    |       p4/c4/p2/c2/nftClass |    (p5,c5)     |    (p6,c6)     | p6/c6/p4/c4/p2/c2/nftClass |    A    |
//|    A    | p6/c6/p4/c4/p2/c2/nftClass |    (p6,c6)     |    (p5,c5)     |       p4/c4/p2/c2/nftClass |    C    |
//|    C    |       p4/c4/p2/c2/nftClass |    (p4,c4)     |    (p3,c3)     |             p2/c2/nftClass |    B    |
//|    B    |             p2/c2/nftClass |    (p2,c2)     |    (p1,c1)     |                   nftClass |    A    |
//
func (k Keeper) SendTransfer(
	ctx sdk.Context,
	sourcePort,
	sourceChannel,
	classID string,
	tokenIDs []string,
	sender sdk.AccAddress,
	receiver string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) error {
	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return sdkerrors.Wrapf(
			channeltypes.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", sourcePort, sourceChannel,
		)
	}

	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	// See spec for this logic: https://github.com/cosmos/ibc/blob/master/spec/app/ics-721-nft-transfer/README.md#packet-relay
	packet, err := k.createOutgoingPacket(ctx,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		classID,
		tokenIDs,
		sender,
		receiver,
		sequence,
		timeoutHeight,
		timeoutTimestamp,
	)
	if err != nil {
		return err
	}

	if err := k.ics4Wrapper.SendPacket(ctx, channelCap, packet); err != nil {
		return err
	}

	defer func() {
		labels := []metrics.Label{
			telemetry.NewLabel(coretypes.LabelDestinationPort, destinationPort),
			telemetry.NewLabel(coretypes.LabelDestinationChannel, destinationChannel),
		}

		telemetry.SetGaugeWithLabels(
			[]string{"tx", "msg", "ibc", "nft-transfer"},
			float32(len(tokenIDs)),
			[]metrics.Label{telemetry.NewLabel("class_id", classID)},
		)

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", types.ModuleName, "send"},
			1,
			labels,
		)
	}()
	return nil
}

// OnRecvPacket processes a cross chain fungible token transfer. If the
// sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// unescrowed and sent to the receiving address.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet,
	data types.NonFungibleTokenPacketData) error {

	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return err
	}

	// See spec for this logic: https://github.com/cosmos/ibc/blob/master/spec/app/ics-721-nft-transfer/README.md#packet-relay
	return k.processReceivedPacket(ctx, packet, data)
}

// OnAcknowledgementPacket responds to the the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement
// was a success then nothing occurs. If the acknowledgement failed, then
// the sender is refunded their tokens using the refundPacketToken function.
func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData, ack channeltypes.Acknowledgement) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		return k.refundPacketToken(ctx, packet, data)
	default:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	}
}

// OnTimeoutPacket refunds the sender since the original packet sent was
// never received and has been timed out.
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData) error {
	return k.refundPacketToken(ctx, packet, data)
}

// refundPacketToken will unescrow and send back the tokens back to sender
// if the sending chain was the source chain. Otherwise, the sent tokens
// were burnt in the original send so new tokens are minted and sent to
// the sending address.
func (k Keeper) refundPacketToken(ctx sdk.Context, packet channeltypes.Packet, data types.NonFungibleTokenPacketData) error {
	if types.IsAwayFromOrigin(packet.GetSourcePort(),
		packet.GetSourceChannel(), data.ClassId) {
		for _, tokenID := range data.TokenIds {
			if err := k.nftKeeper.Transfer(ctx, data.ClassId, tokenID, data.Sender); err != nil {
				return err
			}
		}
		return nil
	}

	classTrace := types.ParseClassTrace(data.ClassId)
	voucherClassID := classTrace.IBCClassID()
	for i, tokenID := range data.TokenIds {
		if err := k.nftKeeper.Mint(ctx,
			voucherClassID, tokenID, data.TokenUris[i], data.Sender); err != nil {
			return err
		}
	}
	return nil
}

// createOutgoingPacket will escrow the tokens to escrow account
// if the token was away from origin chain . Otherwise, the sent tokens
// were burnt in the sending chain and will unescrow the token to receiver
// in the destination chain
func (k Keeper) createOutgoingPacket(ctx sdk.Context,
	sourcePort,
	sourceChannel,
	destinationPort,
	destinationChannel,
	classID string,
	tokenIDs []string,
	sender sdk.AccAddress,
	receiver string,
	sequence uint64,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) (channeltypes.Packet, error) {
	class, exist := k.nftKeeper.GetClass(ctx, classID)
	if !exist {
		return channeltypes.Packet{}, sdkerrors.Wrap(types.ErrInvalidClassID, "classId not exist")
	}

	var (
		// NOTE: class and hex hash correctness checked during msg.ValidateBasic
		fullClassPath = classID
		err           error
		tokenURIs     []string
	)

	// deconstruct the token denomination into the denomination trace info
	// to determine if the sender is the source chain
	if strings.HasPrefix(classID, "ibc/") {
		fullClassPath, err = k.ClassPathFromHash(ctx, classID)
		if err != nil {
			return channeltypes.Packet{}, err
		}
	}

	isAwayFromOrigin := types.IsAwayFromOrigin(sourcePort,
		sourceChannel, fullClassPath)

	for _, tokenID := range tokenIDs {
		nft, exist := k.nftKeeper.GetNFT(ctx, classID, tokenID)
		if !exist {
			return channeltypes.Packet{}, sdkerrors.Wrap(types.ErrInvalidTokenID, "tokenId not exist")
		}
		tokenURIs = append(tokenURIs, nft.GetUri())

		owner := k.nftKeeper.GetOwner(ctx, classID, tokenID)
		if !sender.Equals(owner) {
			return channeltypes.Packet{}, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "not token owner")
		}

		if !isAwayFromOrigin {
			if err := k.nftKeeper.Burn(ctx, classID, tokenID); err != nil {
				return channeltypes.Packet{}, err
			}
			continue
		}

		// create the escrow address for the tokens
		escrowAddress := types.GetEscrowAddress(sourcePort, sourceChannel)
		if err := k.nftKeeper.Transfer(ctx, classID, tokenID, escrowAddress.String()); err != nil {
			return channeltypes.Packet{}, err
		}
	}

	packetData := types.NewNonFungibleTokenPacketData(
		fullClassPath, class.GetUri(), tokenIDs, tokenURIs, sender.String(), receiver,
	)

	return channeltypes.NewPacket(
		packetData.GetBytes(),
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		timeoutHeight,
		timeoutTimestamp,
	), nil
}

// processReceivedPacket will mint the tokens to receiver account
// if the token was away from origin chain . Otherwise, the sent tokens
// were burnt in the sending chain and will unescrow the token to receiver
// in the destination chain
func (k Keeper) processReceivedPacket(ctx sdk.Context, packet channeltypes.Packet,
	data types.NonFungibleTokenPacketData) error {
	if types.IsAwayFromOrigin(packet.GetSourcePort(), packet.GetSourceChannel(), data.ClassId) {
		// since SendPacket did not prefix the classID, we must prefix classID here
		classPrefix := types.GetClassPrefix(packet.GetDestPort(), packet.GetDestChannel())
		// NOTE: sourcePrefix contains the trailing "/"
		prefixedClassID := classPrefix + data.ClassId

		// construct the class trace from the full raw classID
		classTrace := types.ParseClassTrace(prefixedClassID)
		if !k.HasClassTrace(ctx, classTrace.Hash()) {
			k.SetClassTrace(ctx, classTrace)
		}

		voucherClassID := classTrace.IBCClassID()
		if !k.nftKeeper.HasClass(ctx, voucherClassID) {
			if err := k.nftKeeper.SaveClass(ctx, voucherClassID, data.ClassUri); err != nil {
				return err
			}
		}
		for i, tokenID := range data.TokenIds {
			if err := k.nftKeeper.Mint(ctx,
				voucherClassID, tokenID, data.TokenUris[i], data.Receiver); err != nil {
				return err
			}
		}
		return nil
	}

	// If the token moves in the direction of back to origin,
	// we need to unescrow the token and transfer it to the receiver

	// we should remove the prefix. For example:
	// p6/c6/p4/c4/p2/c2/nftClas -> p4/c4/p2/c2/nftClass
	unprefixedClassID := types.RemoveClassPrefix(packet.GetSourcePort(),
		packet.GetSourceChannel(), data.ClassId)
	voucherClassID := types.ParseClassTrace(unprefixedClassID).IBCClassID()
	for _, tokenID := range data.TokenIds {
		if err := k.nftKeeper.Transfer(ctx,
			voucherClassID, tokenID, data.Receiver); err != nil {
			return err
		}
	}
	return nil
}
