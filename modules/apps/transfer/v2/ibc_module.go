package v2

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
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

	if _, ackErr = im.keeper.OnRecvPacket(
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
		ack = channeltypes.NewErrorAcknowledgement(fmt.Errorf("forwarding not yet supported"))
		return types.RecvPacketResult{
			Status:          types.PacketStatus_Failure,
			Acknowledgement: ack.Acknowledgement(),
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

func (im *IBCModule) OnTimeoutPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, payload types.Payload, relayer sdk.AccAddress) error {
	data, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
	if err != nil {
		return err
	}

	// refund tokens
	if err := im.keeper.OnTimeoutPacket(ctx, payload.SourcePort, sourceChannel, data); err != nil {
		return err
	}

	// TODO: handle forwarding

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

	// TODO: handle forwarding

	return im.keeper.EmitOnAcknowledgementPacketEvent(ctx, data, ack)
}
