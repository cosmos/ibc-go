package controller

import (
	"errors"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v9/modules/core/exported"
)

var (
	_ porttypes.Middleware            = (*IBCMiddleware)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCMiddleware)(nil)
	_ porttypes.UpgradableModule      = (*IBCMiddleware)(nil)
)

// IBCMiddleware implements the ICS26 callbacks for the fee middleware given the
// ICA controller keeper and the underlying application.
type IBCMiddleware struct {
	app    porttypes.ClassicIBCModule
	keeper keeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the associated keeper.
// The underlying application is set to nil and authentication is assumed to
// be performed by a Cosmos SDK module that sends messages to controller message server.
func NewIBCMiddleware(k keeper.Keeper) IBCMiddleware {
	return IBCMiddleware{
		app:    nil,
		keeper: k,
	}
}

// NewIBCMiddlewareWithAuth creates a new IBCMiddleware given the associated keeper and underlying application
func NewIBCMiddlewareWithAuth(app porttypes.ClassicIBCModule, k keeper.Keeper) IBCMiddleware {
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
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	if !im.keeper.GetParams(ctx).ControllerEnabled {
		return "", types.ErrControllerSubModuleDisabled
	}

	version, err := im.keeper.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
	if err != nil {
		return "", err
	}
	return version, nil
}

// OnChanOpenTry implements the IBCMiddleware interface
func (IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	return "", errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
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
	if !im.keeper.GetParams(ctx).ControllerEnabled {
		return types.ErrControllerSubModuleDisabled
	}

	if err := im.keeper.OnChanOpenAck(ctx, portID, channelID, counterpartyVersion); err != nil {
		return err
	}

	return nil
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel handshake must be initiated by controller chain")
}

// OnChanCloseInit implements the IBCMiddleware interface
func (IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for interchain account channels
	return errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	if err := im.keeper.OnChanCloseConfirm(ctx, portID, channelID); err != nil {
		return err
	}

	connectionID, err := im.keeper.GetConnectionID(ctx, portID, channelID)
	if err != nil {
		return err
	}

	if im.app != nil && im.keeper.IsMiddlewareEnabled(ctx, portID, connectionID) {
		return im.app.OnChanCloseConfirm(ctx, portID, channelID)
	}

	return nil
}

// OnSendPacket implements the IBCModule interface.
func (im IBCMiddleware) OnSendPacket(
	ctx sdk.Context,
	portID string,
	channelID string,
	sequence uint64,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
	signer sdk.AccAddress,
) error {
	if !im.keeper.GetParams(ctx).ControllerEnabled {
		return types.ErrControllerSubModuleDisabled
	}

	controllerPortID, err := icatypes.NewControllerPortID(signer.String())
	if err != nil {
		return err
	}

	if controllerPortID != portID {
		return errorsmod.Wrap(ibcerrors.ErrUnauthorized, "signer is not owner of interchain account channel")
	}

	connectionID, err := im.keeper.GetConnectionID(ctx, portID, channelID)
	if err != nil {
		return err
	}

	activeChannelID, found := im.keeper.GetOpenActiveChannel(ctx, connectionID, portID)
	if !found {
		return errorsmod.Wrapf(icatypes.ErrActiveChannelNotFound, "failed to retrieve active channel on connection %s for port %s", connectionID, portID)
	}

	if activeChannelID != channelID {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "active channel ID does not match provided channelID. expected %s, got %s", activeChannelID, channelID)
	}

	var icaPacketData icatypes.InterchainAccountPacketData
	if err := icaPacketData.UnmarshalJSON(data); err != nil {
		return err
	}

	if err := icaPacketData.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "invalid interchain account packet data")
	}

	return nil
}

// OnRecvPacket implements the IBCMiddleware interface
func (IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	_ string,
	packet channeltypes.Packet,
	_ sdk.AccAddress,
) ibcexported.Acknowledgement {
	err := errorsmod.Wrapf(icatypes.ErrInvalidChannelFlow, "cannot receive packet on controller chain")
	ack := channeltypes.NewErrorAcknowledgement(err)
	keeper.EmitAcknowledgementEvent(ctx, packet, ack, err)
	return ack
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	if !im.keeper.GetParams(ctx).ControllerEnabled {
		return types.ErrControllerSubModuleDisabled
	}

	connectionID, err := im.keeper.GetConnectionID(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if err != nil {
		return err
	}

	// call underlying app's OnAcknowledgementPacket callback.
	if im.app != nil && im.keeper.IsMiddlewareEnabled(ctx, packet.GetSourcePort(), connectionID) {
		return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
	}

	return nil
}

// OnTimeoutPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	if !im.keeper.GetParams(ctx).ControllerEnabled {
		return types.ErrControllerSubModuleDisabled
	}

	if err := im.keeper.OnTimeoutPacket(ctx, packet); err != nil {
		return err
	}

	connectionID, err := im.keeper.GetConnectionID(ctx, packet.GetSourcePort(), packet.GetSourceChannel())
	if err != nil {
		return err
	}

	if im.app != nil && im.keeper.IsMiddlewareEnabled(ctx, packet.GetSourcePort(), connectionID) {
		return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
	}

	return nil
}

// OnChanUpgradeInit implements the IBCModule interface
func (im IBCMiddleware) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	if !im.keeper.GetParams(ctx).ControllerEnabled {
		return "", types.ErrControllerSubModuleDisabled
	}

	return im.keeper.OnChanUpgradeInit(ctx, portID, channelID, proposedOrder, proposedConnectionHops, proposedVersion)
}

// OnChanUpgradeTry implements the IBCModule interface
func (IBCMiddleware) OnChanUpgradeTry(_ sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	return "", errorsmod.Wrap(icatypes.ErrInvalidChannelFlow, "channel upgrade handshake must be initiated by controller chain")
}

// OnChanUpgradeAck implements the IBCModule interface
func (im IBCMiddleware) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	if !im.keeper.GetParams(ctx).ControllerEnabled {
		return types.ErrControllerSubModuleDisabled
	}

	return im.keeper.OnChanUpgradeAck(ctx, portID, channelID, counterpartyVersion)
}

// OnChanUpgradeOpen implements the IBCModule interface
func (IBCMiddleware) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	panic(errors.New("WriteAcknowledgement not supported for ICA controller module"))
}

// GetAppVersion returns the interchain accounts metadata.
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.keeper.GetAppVersion(ctx, portID, channelID)
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes
// into an InterchainAccountPacketData. This function implements the optional
// PacketDataUnmarshaler interface required for ADR 008 support.
func (im IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (interface{}, string, error) {
	var data icatypes.InterchainAccountPacketData
	err := data.UnmarshalJSON(bz)
	if err != nil {
		return nil, "", err
	}

	version, ok := im.GetAppVersion(ctx, portID, channelID)
	if !ok {
		return nil, "", errorsmod.Wrapf(ibcerrors.ErrNotFound, "app version not found for port %s and channel %s", portID, channelID)
	}

	return data, version, nil
}

// WrapVersion returns the wrapped version based on the provided version string and underlying application version.
// TODO: decide how we want to handle the underlying app. For now I made it backwards compatible.
// https://github.com/cosmos/ibc-go/issues/7063
func (IBCMiddleware) WrapVersion(cbVersion, underlyingAppVersion string) string {
	// ignore underlying app version
	return cbVersion
}

// UnwrapVersionUnsafe returns the version. Interchain accounts does not wrap versions.
func (IBCMiddleware) UnwrapVersionUnsafe(version string) (string, string, error) {
	// ignore underlying app version
	return version, "", nil
}
