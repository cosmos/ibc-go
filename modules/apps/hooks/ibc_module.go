package ibc_hooks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v5/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v5/modules/core/exported"
)

var (
	_ porttypes.Middleware = &IBCMiddleware{}
)

type IBCMiddleware struct {
	App            porttypes.IBCModule
	ICS4Middleware *ICS4Middleware

	// Hooks
	Hooks IBCAppHooks
}

func NewIBCMiddleware(app porttypes.IBCModule, ics4 *ICS4Middleware) IBCMiddleware {
	return IBCMiddleware{
		App:            app,
		ICS4Middleware: ics4,
	}
}

func (im IBCMiddleware) WithHooks(hooks IBCAppHooks) IBCMiddleware {
	im.Hooks = hooks
	return im
}

// OnChanOpenInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenInitOverride); ok {
		return hook.OnChanOpenInitOverride(im, ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)
	}

	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenInitBefore); ok {
		hook.OnChanOpenInitBeforeHook(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)
	}

	result, err := im.App.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)

	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenInitAfter); ok {
		hook.OnChanOpenInitAfterHook(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version, result, err)
	}
	return result, err
}

// OnChanOpenTry implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenTryOverride); ok {
		return hook.OnChanOpenTryOverride(im, ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)
	}

	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenTryBefore); ok {
		hook.OnChanOpenTryBeforeHook(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)
	}

	version, err := im.App.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)

	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenTryAfter); ok {
		hook.OnChanOpenTryAfterHook(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion, version, err)
	}
	return version, err
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenAckOverride); ok {
		return hook.OnChanOpenAckOverride(im, ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	}

	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenAckBefore); ok {
		hook.OnChanOpenAckBeforeHook(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	}
	err := im.App.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenAckAfter); ok {
		hook.OnChanOpenAckAfterHook(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion, err)
	}

	return err
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenConfirmOverride); ok {
		return hook.OnChanOpenConfirmOverride(im, ctx, portID, channelID)
	}

	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenConfirmBefore); ok {
		hook.OnChanOpenConfirmBeforeHook(ctx, portID, channelID)
	}
	err := im.App.OnChanOpenConfirm(ctx, portID, channelID)
	if hook, ok := im.Hooks.(IBCAppHooksOnChanOpenConfirmAfter); ok {
		hook.OnChanOpenConfirmAfterHook(ctx, portID, channelID, err)
	}
	return err
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Here we can remove the limits when a new channel is closed. For now, they can remove them  manually on the contract
	if hook, ok := im.Hooks.(IBCAppHooksOnChanCloseInitOverride); ok {
		return hook.OnChanCloseInitOverride(im, ctx, portID, channelID)
	}

	if hook, ok := im.Hooks.(IBCAppHooksOnChanCloseInitBefore); ok {
		hook.OnChanCloseInitBeforeHook(ctx, portID, channelID)
	}
	err := im.App.OnChanCloseInit(ctx, portID, channelID)
	if hook, ok := im.Hooks.(IBCAppHooksOnChanCloseInitAfter); ok {
		hook.OnChanCloseInitAfterHook(ctx, portID, channelID, err)
	}

	return err
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Here we can remove the limits when a new channel is closed. For now, they can remove them  manually on the contract
	if hook, ok := im.Hooks.(IBCAppHooksOnChanCloseConfirmOverride); ok {
		return hook.OnChanCloseConfirmOverride(im, ctx, portID, channelID)
	}

	if hook, ok := im.Hooks.(IBCAppHooksOnChanCloseConfirmBefore); ok {
		hook.OnChanCloseConfirmBeforeHook(ctx, portID, channelID)
	}
	err := im.App.OnChanCloseConfirm(ctx, portID, channelID)
	if hook, ok := im.Hooks.(IBCAppHooksOnChanCloseConfirmAfter); ok {
		hook.OnChanCloseConfirmAfterHook(ctx, portID, channelID, err)
	}

	return err
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return im.App.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	if hook, ok := im.Hooks.(IBCAppHooksOnTimeoutPacketOverride); ok {
		return hook.OnTimeoutPacketOverride(im, ctx, packet, relayer)
	}

	if hook, ok := im.Hooks.(IBCAppHooksOnTimeoutPacketBefore); ok {
		hook.OnTimeoutPacketBeforeHook(ctx, packet, relayer)
	}
	err := im.App.OnTimeoutPacket(ctx, packet, relayer)
	if hook, ok := im.Hooks.(IBCAppHooksOnTimeoutPacketAfter); ok {
		hook.OnTimeoutPacketAfterHook(ctx, packet, relayer, err)
	}

	return err
}

// OnRecvPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	if hook, ok := im.Hooks.(IBCAppHooksOnRecvPacketOverride); ok {
		return hook.OnRecvPacketOverride(im, ctx, packet, relayer)
	}

	if hook, ok := im.Hooks.(IBCAppHooksOnRecvPacketBefore); ok {
		hook.OnRecvPacketBeforeHook(ctx, packet, relayer)
	}

	ack := im.App.OnRecvPacket(ctx, packet, relayer)

	if hook, ok := im.Hooks.(IBCAppHooksOnRecvPacketAfter); ok {
		hook.OnRecvPacketAfterHook(ctx, packet, relayer, ack)
	}

	return ack
}

// SendPacket implements the ICS4 Wrapper interface
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
) error {
	if hook, ok := im.Hooks.(IBCAppHooksSendPacketOverride); ok {
		return hook.SendPacketOverride(im, ctx, chanCap, packet)
	}

	if hook, ok := im.Hooks.(IBCAppHooksSendPacketBefore); ok {
		hook.SendPacketBeforeHook(ctx, chanCap, packet)
	}
	err := im.ICS4Middleware.SendPacket(ctx, chanCap, packet)
	if hook, ok := im.Hooks.(IBCAppHooksSendPacketAfter); ok {
		hook.SendPacketAfterHook(ctx, chanCap, packet, err)
	}

	return err
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	if hook, ok := im.Hooks.(IBCAppHooksWriteAcknowledgementOverride); ok {
		return hook.WriteAcknowledgementOverride(im, ctx, chanCap, packet, ack)
	}

	if hook, ok := im.Hooks.(IBCAppHooksWriteAcknowledgementBefore); ok {
		hook.WriteAcknowledgementBeforeHook(ctx, chanCap, packet, ack)
	}
	err := im.ICS4Middleware.WriteAcknowledgement(ctx, chanCap, packet, ack)
	if hook, ok := im.Hooks.(IBCAppHooksWriteAcknowledgementAfter); ok {
		hook.WriteAcknowledgementAfterHook(ctx, chanCap, packet, ack, err)
	}

	return err
}

func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	if hook, ok := im.Hooks.(IBCAppHooksGetAppVersionOverride); ok {
		return hook.GetAppVersionOverride(im, ctx, portID, channelID)
	}

	if hook, ok := im.Hooks.(IBCAppHooksGetAppVersionBefore); ok {
		hook.GetAppVersionBeforeHook(ctx, portID, channelID)
	}
	version, err := im.ICS4Middleware.GetAppVersion(ctx, portID, channelID)
	if hook, ok := im.Hooks.(IBCAppHooksGetAppVersionAfter); ok {
		hook.GetAppVersionAfterHook(ctx, portID, channelID, version, err)
	}

	return version, err
}
