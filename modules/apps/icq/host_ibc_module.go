package icq

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/icq/keeper"
	"github.com/cosmos/ibc-go/v3/modules/apps/icq/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// HostIBCModule implements the ICS26 interface for interchain query host chains
type HostIBCModule struct {
	keeper keeper.Keeper
}

// NewIBCModule creates a new IBCModule given the associated keeper
func NewIBCModule(k keeper.Keeper) HostIBCModule {
	return HostIBCModule{
		keeper: k,
	}
}

// OnChanOpenInit implements the IBCModule interface
func (im HostIBCModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return "", sdkerrors.Wrap(types.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanOpenTry implements the IBCModule interface
func (im HostIBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if !im.keeper.IsHostEnabled(ctx) {
		return "", types.ErrHostDisabled
	}

	return im.keeper.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCModule interface
func (im HostIBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return sdkerrors.Wrap(types.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanOpenAck implements the IBCModule interface
func (im HostIBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	if !im.keeper.IsHostEnabled(ctx) {
		return types.ErrHostDisabled
	}

	return im.keeper.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface
func (im HostIBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for interchain query channels
	return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (im HostIBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.keeper.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCModule interface
func (im HostIBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	_ sdk.AccAddress,
) ibcexported.Acknowledgement {
	if !im.keeper.IsHostEnabled(ctx) {
		return types.NewErrorAcknowledgement(types.ErrHostDisabled)
	}

	txResponse, err := im.keeper.OnRecvPacket(ctx, packet)
	if err != nil {
		// Emit an event including the error msg
		keeper.EmitWriteErrorAcknowledgementEvent(ctx, packet, err)

		return types.NewErrorAcknowledgement(err)
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return channeltypes.NewResultAcknowledgement(txResponse)
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im HostIBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return sdkerrors.Wrap(types.ErrInvalidChannelFlow, "cannot receive acknowledgement on a host channel end, a host chain does not send a packet over the channel")
}

// OnTimeoutPacket implements the IBCModule interface
func (im HostIBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return sdkerrors.Wrap(types.ErrInvalidChannelFlow, "cannot cause a packet timeout on a host channel end, a host chain does not send a packet over the channel")
}
