package transfer

import (
	errorsmod "cosmossdk.io/errors"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/events"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var (
	_ porttypes.IBCModuleV2 = (*IBCModuleV2)(nil)
)

// IBCModuleV2 implements the ICS26 interface for transfer given the transfer keeper.
type IBCModuleV2 struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModuleV2(k keeper.Keeper) IBCModuleV2 {
	return IBCModuleV2{
		keeper: k,
	}
}

// OnSendPacketV2 implements the IBCModuleV2 interface.
func (im IBCModuleV2) OnSendPacketV2(
	ctx sdk.Context,
	portID string,
	channelID string,
	sequence uint64,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	payload channeltypes.Payload,
	signer sdk.AccAddress,
) error {
	if !im.keeper.GetParams(ctx).SendEnabled {
		return types.ErrSendDisabled
	}
	if im.keeper.IsBlockedAddr(signer) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to send funds", signer)
	}

	data, err := types.UnmarshalPacketData(payload.Value, payload.Version)
	if err != nil {
		return err
	}

	// If the ics20version is V1, we can't have multiple tokens nor forwarding info.
	// However, we do not need to check it here, as a packet containing that data would
	// fail the unmarshaling above, where if ics20version == types.V1 we first unmarshal
	// into a V1 packet and then convert.

	if data.Sender != signer.String() {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "invalid signer address: expected %s, got %s", data.Sender, signer)
	}

	return im.keeper.OnSendPacket(ctx, portID, channelID, data, signer)
}

// OnRecvPacketV2 implements the IBCModule interface. A successful acknowledgement
// is returned if the packet data is successfully decoded and the receive application
// logic returns without error.
// A nil acknowledgement may be returned when using the packet forwarding feature. This signals to core IBC that the acknowledgement will be written asynchronously.
func (im IBCModuleV2) OnRecvPacketV2(
	ctx sdk.Context,
	packet channeltypes.PacketV2,
	payload channeltypes.Payload,
	relayer sdk.AccAddress,
) channeltypes.RecvPacketResult {
	var (
		ackErr error
		data   types.FungibleTokenPacketDataV2
	)

	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	// we are explicitly wrapping this emit event call in an anonymous function so that
	// the packet data is evaluated after it has been assigned a value.
	defer func() {
		events.EmitOnRecvPacketEvent(ctx, data, ack, ackErr)
	}()

	data, ackErr = types.UnmarshalPacketData(payload.Value, payload.Version)
	if ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		return channeltypes.RecvPacketResult{
			Status:          channeltypes.PacketStatus_Failure,
			Acknowledgement: ack.Acknowledgement(),
		}
	}

	if ackErr = im.keeper.OnRecvPacketV2(ctx, packet, data); ackErr != nil {
		ack = channeltypes.NewErrorAcknowledgement(ackErr)
		im.keeper.Logger(ctx).Error(fmt.Sprintf("%s sequence %d", ackErr.Error(), packet.Sequence))
		return channeltypes.RecvPacketResult{
			Status:          channeltypes.PacketStatus_Failure,
			Acknowledgement: ack.Acknowledgement(),
		}
	}

	im.keeper.Logger(ctx).Info("successfully handled ICS-20 packet", "sequence", packet.Sequence)

	if data.HasForwarding() {
		// NOTE: acknowledgement will be written asynchronously
		return channeltypes.RecvPacketResult{
			Status: channeltypes.PacketStatus_Async,
		}
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return channeltypes.RecvPacketResult{
		Status:          channeltypes.PacketStatus_Success,
		Acknowledgement: ack.Acknowledgement(),
	}
}

func (im IBCModuleV2) OnAcknowledgementPacketV2(
	ctx sdk.Context,
	packet channeltypes.PacketV2,
	payload channeltypes.Payload,
	recvPacketResult channeltypes.RecvPacketResult,
	relayer sdk.AccAddress,
) error {
	// TODO: use the recvPacketResult directly, don't need to unmarshal the acknowledgement
	var ack channeltypes.Acknowledgement
	if err := types.ModuleCdc.UnmarshalJSON(recvPacketResult.Acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %v", err)
	}

	data, err := types.UnmarshalPacketData(payload.Value, payload.Version)
	if err != nil {
		return err
	}

	if err := im.keeper.OnAcknowledgementPacketV2(ctx, packet, data, ack); err != nil {
		return err
	}

	events.EmitOnAcknowledgementPacketEvent(ctx, data, ack)

	return nil
}

func (im IBCModuleV2) OnTimeoutPacketV2(
	ctx sdk.Context,
	packet channeltypes.PacketV2,
	payload channeltypes.Payload,
	relayer sdk.AccAddress,
) error {
	data, err := types.UnmarshalPacketData(payload.Value, payload.Version)
	if err != nil {
		return err
	}

	// refund tokens
	if err := im.keeper.OnTimeoutPacketV2(ctx, packet, data); err != nil {
		return err
	}

	events.EmitOnTimeoutEvent(ctx, data)
	return nil
}
