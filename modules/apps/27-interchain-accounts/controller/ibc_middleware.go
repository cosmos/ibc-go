package controller

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"
)

var _ porttypes.Middleware = &IBCMiddleware{}

// IBCMiddleware implements the ICS26 callbacks for the fee middleware given the
// ICA controller keeper and the underlying application.
type IBCMiddleware struct {
	app    porttypes.IBCModule
	keeper keeper.Keeper
}

// IBCMiddleware creates a new IBCMiddleware given the associated keeper and underlying application
func NewIBCMiddleware(app porttypes.IBCModule, k keeper.Keeper) IBCMiddleware {
	return IBCMiddleware{
		app:    app,
		keeper: k,
	}
}

// OnChanOpenInit implements the IBCMiddleware interface
//
// Interchain Accounts is implemented to act as middleware for connected authentication modules on
// the controller side. The connected modules may not change the controller side portID or
// version. They will be allowed to perform custom logic without changing
// the parameters stored within a channel struct.
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	if !im.keeper.IsControllerEnabled(ctx) {
		return "", types.ErrControllerSubModuleDisabled
	}

	version, err := im.keeper.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
	if err != nil {
		return "", err
	}

	// call underlying app's OnChanOpenInit callback with the passed in version
	// the version returned is discarded as the ica-auth module does not have permission to edit the version string.
	// ics27 will always return the version string containing the Metadata struct which is created during the `RegisterInterchainAccount` call.
	if _, err := im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version); err != nil {
		return "", err
	}

	return version, nil
}

// OnChanOpenTry implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return "", sdkerrors.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanOpenAck implements the IBCMiddleware interface
//
// Interchain Accounts is implemented to act as middleware for connected authentication modules on
// the controller side. The connected modules may not change the portID or
// version. They will be allowed to perform custom logic without changing
// the parameters stored within a channel struct.
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	if !im.keeper.IsControllerEnabled(ctx) {
		return types.ErrControllerSubModuleDisabled
	}

	if err := im.keeper.OnChanOpenAck(ctx, portID, channelID, counterpartyVersion); err != nil {
		return err
	}

	// call underlying app's OnChanOpenAck callback with the counterparty app version.
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return sdkerrors.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for interchain account channels
	return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return im.keeper.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	_ sdk.AccAddress,
) ibcexported.Acknowledgement {
	err := sdkerrors.Wrapf(icatypes.ErrInvalidChannelFlow, "cannot receive packet on controller chain")
	ack := channeltypes.NewErrorAcknowledgement(err)
	keeper.EmitAcknowledgementEvent(ctx, packet, ack, err)
	return ack
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	if !im.keeper.IsControllerEnabled(ctx) {
		return types.ErrControllerSubModuleDisabled
	}

	// call underlying app's OnAcknowledgementPacket callback.
	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	if !im.keeper.IsControllerEnabled(ctx) {
		return types.ErrControllerSubModuleDisabled
	}

	if err := im.keeper.OnTimeoutPacket(ctx, packet); err != nil {
		return err
	}

	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

// SendPacket implements the ICS4 Wrapper interface
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
) error {
	panic("SendPacket not supported for ICA controller module. Please use SendTx")
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	panic("WriteAcknowledgement not supported for ICA controller module")
}

// GetAppVersion returns the interchain accounts metadata.
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.keeper.GetAppVersion(ctx, portID, channelID)
}
