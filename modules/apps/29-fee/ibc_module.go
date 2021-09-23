package fee

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	
	"github.com/cosmos/ibc-go/modules/apps/29-fee/keeper"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/modules/core/exported"
)

// IBCModule implements the ICS26 callbacks for the fee middleware given the fee keeper and the underlying application.
type IBCModule struct {
	keeper keeper.Keeper
	app    porttypes.IBCModule
}

// NewIBCModule creates a new IBCModule given the keeper and underlying application
func NewIBCModule(k keeper.Keeper, app porttypes.IBCModule) IBCModule {
	return IBCModule{
		keeper: k,
		app:    app,
	}
}

// OnChanOpenInit implements the IBCModule interface
func (im IBCModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) error {
	feeVersion, appVersion := channeltypes.SplitChannelVersion(version)
	if feeVersion != "" {
		if feeVersion != types.Version {
			return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected: %s, got: %s", types.Version, feeVersion)
		}

		im.keeper.SetFeeEnabled(ctx, portID, channelID)
	}

	// call underlying app's OnChanOpenInit callback with the appVersion
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID,
		chanCap, counterparty, appVersion)
}

// OnChanOpenTry implements the IBCModule interface
func (im IBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version,
	counterpartyVersion string,
) error {
	feeVersion, appVersion := channeltypes.SplitChannelVersion(version)
	cpFeeVersion, cpAppVersion := channeltypes.SplitChannelVersion(counterpartyVersion)

	if feeVersion != "" || cpFeeVersion != "" {
		if feeVersion != types.Version {
			return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected: %s, got: %s", types.Version, feeVersion)
		}
		if cpFeeVersion != feeVersion {
			return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected counterparty version: %s, got: %s", types.Version, cpFeeVersion)
		}

		im.keeper.SetFeeEnabled(ctx, portID, channelID)
	} else if im.keeper.IsFeeEnabled(ctx, portID, channelID) {
		// return error if a previous ChanInit set fee enabled but subsequent OpenTry does not have fee enabled
		return sdkerrors.Wrapf(types.ErrInvalidVersion, "previous INIT call (crossing hellos) had fee version set but OpenTry call does not set fee version: %s", version)
	}
	// call underlying app's OnChanOpenTry callback with the app versions
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID,
		chanCap, counterparty, appVersion, cpAppVersion)
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyVersion string,
) error {
	cpAppVersion := counterpartyVersion
	if im.keeper.IsFeeEnabled(ctx, portID, channelID) {
		var cpFeeVersion string
		cpFeeVersion, cpAppVersion = channeltypes.SplitChannelVersion(counterpartyVersion)

		if cpFeeVersion != types.Version {
			return sdkerrors.Wrapf(types.ErrInvalidVersion, "expected counterparty version: %s, got: %s", types.Version, cpFeeVersion)
		}
	}
	// call underlying app's OnChanOpenAck callback with the counterparty app version.
	return im.app.OnChanOpenAck(ctx, portID, channelID, cpAppVersion)
}

// OnChanOpenConfirm implements the IBCModule interface
func (im IBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// call underlying app's OnChanOpenConfirm callback.
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface
func (im IBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// TODO: Unescrow all remaining funds for unprocessed packets
	im.keeper.DeleteFeeEnabled(ctx, portID, channelID)
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// TODO: Unescrow all remaining funds for unprocessed packets
	im.keeper.DeleteFeeEnabled(ctx, portID, channelID)
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket implements the IBCModule interface.
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	// TODO: Implement fee specific logic if fee is enabled for the given channel
	return im.app.OnRecvPacket(ctx, packet, relayer)
}

// OnAcknowledgementPacket implements the IBCModule interface
func (im IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// TODO: Implement fee specific logic if fee is enabled for the given channel
	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCModule interface
func (im IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	// TODO: Implement fee specific logic if fee is enabled for the given channel
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}
