package v2

import (
	"context"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	internaltypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	channelv2types "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var _ api.IBCModule = (*IBCModule)(nil)

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper, chanV2Keeper transfertypes.ChannelKeeperV2) *IBCModule {
	return &IBCModule{
		keeper:       k,
		chanV2Keeper: chanV2Keeper,
	}
}

type IBCModule struct {
	keeper       keeper.Keeper
	chanV2Keeper transfertypes.ChannelKeeperV2
}

func (im *IBCModule) OnSendPacket(goCtx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload types.Payload, signer sdk.AccAddress) error {
	data, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
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

	if err := im.keeper.SendTransfer(goCtx, payload.SourcePort, sourceChannel, data.Tokens, signer); err != nil {
		return err
	}

	// TODO: events
	// events.EmitTransferEvent(ctx, sender.String(), receiver, tokens, memo, hops)

	// TODO: telemetry
	// telemetry.ReportTransfer(sourcePort, sourceChannel, destinationPort, destinationChannel, tokens)

	return nil
}

func (im *IBCModule) OnRecvPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload types.Payload, relayer sdk.AccAddress) types.RecvPacketResult {
	var (
		ackErr error
		data   transfertypes.FungibleTokenPacketDataV2
	)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	recvResult := types.RecvPacketResult{
		Status:          types.PacketStatus_Success,
		Acknowledgement: ack.Acknowledgement(),
	}
	// we are explicitly wrapping this emit event call in an anonymous function so that
	// the packet data is evaluated after it has been assigned a value.
	defer func() {
		if err := im.keeper.EmitOnRecvPacketEvent(ctx, data, ack, ackErr); err != nil {
			im.keeper.Logger.Error(fmt.Sprintf("failed to emit %T event", types.EventTypeRecvPacket), "error", err)
		}
	}()

	data, ackErr = transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger.Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), sequence))
		return types.RecvPacketResult{
			Status:          types.PacketStatus_Failure,
			Acknowledgement: ack.Acknowledgement(),
		}
	}

	var receivedCoins sdk.Coins
	if receivedCoins, ackErr = im.keeper.OnRecvPacket(
		ctx,
		data,
		payload.SourcePort,
		sourceChannel,
		payload.DestinationPort,
		destinationChannel,
	); ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger.Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), sequence))
		return types.RecvPacketResult{
			Status:          types.PacketStatus_Failure,
			Acknowledgement: ack.Acknowledgement(),
		}
	}

	im.keeper.Logger.Info("successfully handled ICS-20 packet", "sequence", sequence)

	// TODO: telemetry
	// telemetry.ReportOnRecvPacket(packet, data.Tokens)

	if data.HasForwarding() {
		// we are now sending from the forward escrow address to the final receiver address.
		timeoutTimestamp := uint64(sdkCtx.BlockTime().Add(time.Hour).Unix())
		if err := im.forwardPacket(ctx, destinationChannel, payload.DestinationPort, sequence, data, timeoutTimestamp, receivedCoins); err != nil {
			return types.RecvPacketResult{
				Status:          types.PacketStatus_Failure,
				Acknowledgement: channeltypes.NewErrorAcknowledgement(err).Acknowledgement(),
			}
		}

		// NOTE: acknowledgement will be written asynchronously
		return types.RecvPacketResult{
			Status: types.PacketStatus_Async,
		}
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return recvResult
}

func (im *IBCModule) OnTimeoutPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload types.Payload, relayer sdk.AccAddress) error {
	data, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	// refund tokens
	if err := im.keeper.OnTimeoutPacket(ctx, payload.SourcePort, sourceChannel, data); err != nil {
		return err
	}

	if awaitPacketId, isForwarded := im.keeper.GetForwardV2PacketId(ctx, sourceChannel, sequence); isForwarded {
		// revert the receive of the original packet
		if err := im.keeper.RevertForwardedPacket(ctx, awaitPacketId.PortId, awaitPacketId.ChannelId, data); err != nil {
			return err
		}

		// write an async failed acknowledgement for original received packet since our forwarded packet timed out
		awaitAck := internaltypes.NewForwardTimeoutAcknowledgement(payload.SourcePort, sourceChannel)
		awaitAcknowledgement := channelv2types.NewAcknowledgement(awaitAck.Acknowledgement())
		im.chanV2Keeper.WriteAcknowledgement(ctx, awaitPacketId.ChannelId, awaitPacketId.Sequence, awaitAcknowledgement)

		// delete forwardPacketId
		im.keeper.DeleteForwardV2PacketId(ctx, sourceChannel, sequence)
	}

	return im.keeper.EmitOnTimeoutEvent(ctx, data)
}

func (im *IBCModule) OnAcknowledgementPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, acknowledgement []byte, payload types.Payload, relayer sdk.AccAddress) error {
	var ack channeltypes.Acknowledgement
	if err := transfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	data, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, payload.SourcePort, sourceChannel, data, ack); err != nil {
		return err
	}

	if awaitPacketId, isForwarded := im.keeper.GetForwardV2PacketId(ctx, sourceChannel, sequence); isForwarded {
		var awaitAck channeltypes.Acknowledgement

		switch ack.Response.(type) {
		case *channeltypes.Acknowledgement_Result:
			// Write a successful async ack for awaitPacket
			awaitAck = channeltypes.NewResultAcknowledgement([]byte{byte(1)})
		case *channeltypes.Acknowledgement_Error:
			// the forwarded packet has failed, thus the funds have been refunded to the intermediate address.
			// we must revert the changes that came from successfully receiving the tokens on our chain
			// before propagating the error acknowledgement back to original sender chain
			if err := im.keeper.RevertForwardedPacket(ctx, awaitPacketId.PortId, awaitPacketId.ChannelId, data); err != nil {
				return err
			}

			awaitAck = internaltypes.NewForwardErrorAcknowledgement(payload.SourcePort, sourceChannel, ack)
		default:
			return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected one of [%T, %T], got %T", channeltypes.Acknowledgement_Result{}, channeltypes.Acknowledgement_Error{}, ack.Response)
		}

		// write async acknowledgment for original received packet
		asyncAcknowledgement := channelv2types.NewAcknowledgement(awaitAck.Acknowledgement())
		im.chanV2Keeper.WriteAcknowledgement(ctx, awaitPacketId.ChannelId, awaitPacketId.Sequence, asyncAcknowledgement)

		// delete forwardPacketId
		im.keeper.DeleteForwardV2PacketId(ctx, sourceChannel, sequence)
	}

	return im.keeper.EmitOnAcknowledgementPacketEvent(ctx, data, ack)
}
