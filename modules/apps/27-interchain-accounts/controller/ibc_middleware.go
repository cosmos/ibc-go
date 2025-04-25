package controller

import (
	"errors"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ porttypes.Middleware            = (*IBCMiddleware)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCMiddleware)(nil)
)

// IBCMiddleware implements the ICS26 callbacks for the controller middleware given the
// ICA controller keeper and the underlying application.
type IBCMiddleware struct {
	app    porttypes.IBCModule
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
func NewIBCMiddlewareWithAuth(app porttypes.IBCModule, k keeper.Keeper) IBCMiddleware {
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

	// call underlying app's OnChanOpenInit callback with the passed in version
	// the version returned is discarded as the ica-auth module does not have permission to edit the version string.
	// ics27 will always return the version string containing the Metadata struct which is created during the `RegisterInterchainAccount` call.
	if im.app != nil && im.keeper.IsMiddlewareEnabled(ctx, portID, connectionHops[0]) {
		if _, err := im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version); err != nil {
			return "", err
		}
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

	connectionID, err := im.keeper.GetConnectionID(ctx, portID, channelID)
	if err != nil {
		return err
	}

	// call underlying app's OnChanOpenAck callback with the counterparty app version.
	if im.app != nil && im.keeper.IsMiddlewareEnabled(ctx, portID, connectionID) {
		return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
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

// SendPacket implements the ICS4 Wrapper interface
func (IBCMiddleware) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	panic(errors.New("SendPacket not supported for ICA controller module. Please use SendTx"))
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
func (im IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID string, channelID string, bz []byte) (any, string, error) {
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
