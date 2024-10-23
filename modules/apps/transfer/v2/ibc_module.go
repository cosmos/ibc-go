package v2

import (
	"context"
	errorsmod "cosmossdk.io/errors"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/events"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/v2/keeper"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v9/modules/core/api"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var (
	_ api.IBCModule = (*IBCModule)(nil)
)

// TODO: pass sequence as argument to OnRecvPacket
const sequence = 1

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(keeper keeper.Keeper) IBCModule {
	return IBCModule{
		keeper: keeper,
	}
}

type IBCModule struct {
	keeper keeper.Keeper
}

func (im *IBCModule) OnSendPacket(ctx context.Context, sourceChannel string, destinationChannel string, sequence uint64, data types.Payload, signer sdk.AccAddress) error {
	panic("implement me")
}

func (im *IBCModule) OnRecvPacket(ctx context.Context, sourceChannel string, destinationChannel string, payload types.Payload, relayer sdk.AccAddress) types.RecvPacketResult {
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
		events.EmitOnRecvPacketEvent(ctx, data, ack, ackErr)
	}()

	data, ackErr = transfertypes.UnmarshalPacketData(payload.Value, payload.Version)
	if ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), sequence))
		return types.RecvPacketResult{
			Status:          types.PacketStatus_Failure,
			Acknowledgement: ack.Acknowledgement(),
		}
	}

	if ackErr = im.keeper.OnRecvPacket(ctx, sourceChannel, destinationChannel, payload, data); ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), sequence))
		return types.RecvPacketResult{
			Status:          types.PacketStatus_Failure,
			Acknowledgement: ack.Acknowledgement(),
		}
	}

	im.keeper.Logger(ctx).Info("successfully handled ICS-20 packet", "sequence", sequence)

	if data.HasForwarding() {
		// NOTE: acknowledgement will be written asynchronously
		return types.RecvPacketResult{
			Status: types.PacketStatus_Async,
		}
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return recvResult
}

func (im *IBCModule) OnTimeoutPacket(ctx context.Context, sourceChannel string, destinationChannel string, payload types.Payload, relayer sdk.AccAddress) error {
	panic("implement me")
}

func (im *IBCModule) OnAcknowledgementPacket(ctx context.Context, sourceChannel string, destinationChannel string, payload types.Payload, acknowledgement []byte, relayer sdk.AccAddress) error {
	var ack channeltypes.Acknowledgement
	if err := transfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	data, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version)
	if err != nil {
		return err
	}

	if err := im.keeper.OnAcknowledgementPacket(ctx, payload.SourcePort, sourceChannel, data, ack); err != nil {
		return err
	}

	// TODO: emit events
	//events.EmitOnAcknowledgementPacketEvent(ctx, data, ack)
	return nil
}
