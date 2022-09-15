package ibc_hooks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v5/modules/core/exported"
)

type IBCAppHooks interface {
}

type IBCAppHooksOnChanOpenInitOverride interface {
	OnChanOpenInitOverride(im IBCMiddleware, ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, channelCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string) (string, error)
}
type IBCAppHooksOnChanOpenInitBefore interface {
	OnChanOpenInitBeforeHook(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, channelCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string)
}
type IBCAppHooksOnChanOpenInitAfter interface {
	OnChanOpenInitAfterHook(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, channelCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string, result string, err error)
}

// OnChanOpenTry Hooks
type IBCAppHooksOnChanOpenTryOverride interface {
	OnChanOpenTryOverride(im IBCMiddleware, ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, channelCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error)
}
type IBCAppHooksOnChanOpenTryBefore interface {
	OnChanOpenTryBeforeHook(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, channelCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, counterpartyVersion string)
}
type IBCAppHooksOnChanOpenTryAfter interface {
	OnChanOpenTryAfterHook(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, channelCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, counterpartyVersion string, version string, err error)
}

// OnChanOpenAck Hooks
type IBCAppHooksOnChanOpenAckOverride interface {
	OnChanOpenAckOverride(im IBCMiddleware, ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string) error
}
type IBCAppHooksOnChanOpenAckBefore interface {
	OnChanOpenAckBeforeHook(ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string)
}
type IBCAppHooksOnChanOpenAckAfter interface {
	OnChanOpenAckAfterHook(ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string, err error)
}

// OnChanOpenConfirm Hooks
type IBCAppHooksOnChanOpenConfirmOverride interface {
	OnChanOpenConfirmOverride(im IBCMiddleware, ctx sdk.Context, portID, channelID string) error
}
type IBCAppHooksOnChanOpenConfirmBefore interface {
	OnChanOpenConfirmBeforeHook(ctx sdk.Context, portID, channelID string)
}
type IBCAppHooksOnChanOpenConfirmAfter interface {
	OnChanOpenConfirmAfterHook(ctx sdk.Context, portID, channelID string, err error)
}

// OnChanCloseInit Hooks
type IBCAppHooksOnChanCloseInitOverride interface {
	OnChanCloseInitOverride(im IBCMiddleware, ctx sdk.Context, portID, channelID string) error
}
type IBCAppHooksOnChanCloseInitBefore interface {
	OnChanCloseInitBeforeHook(ctx sdk.Context, portID, channelID string)
}
type IBCAppHooksOnChanCloseInitAfter interface {
	OnChanCloseInitAfterHook(ctx sdk.Context, portID, channelID string, err error)
}

// OnChanCloseConfirm Hooks
type IBCAppHooksOnChanCloseConfirmOverride interface {
	OnChanCloseConfirmOverride(im IBCMiddleware, ctx sdk.Context, portID, channelID string) error
}
type IBCAppHooksOnChanCloseConfirmBefore interface {
	OnChanCloseConfirmBeforeHook(ctx sdk.Context, portID, channelID string)
}
type IBCAppHooksOnChanCloseConfirmAfter interface {
	OnChanCloseConfirmAfterHook(ctx sdk.Context, portID, channelID string, err error)
}

// OnAcknowledgementPacket Hooks
type IBCAppHooksOnAcknowledgementPacketOverride interface {
	OnAcknowledgementPacketOverride(im IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error
}
type IBCAppHooksOnAcknowledgementPacketBefore interface {
	OnAcknowledgementPacketBeforeHook(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress)
}
type IBCAppHooksOnAcknowledgementPacketAfter interface {
	OnAcknowledgementPacketAfterHook(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress, err error)
}

// OnTimeoutPacket Hooks
type IBCAppHooksOnTimeoutPacketOverride interface {
	OnTimeoutPacketOverride(im IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error
}
type IBCAppHooksOnTimeoutPacketBefore interface {
	OnTimeoutPacketBeforeHook(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress)
}
type IBCAppHooksOnTimeoutPacketAfter interface {
	OnTimeoutPacketAfterHook(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress, err error)
}

// OnRecvPacket Hooks
type IBCAppHooksOnRecvPacketOverride interface {
	OnRecvPacketOverride(im IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement
}
type IBCAppHooksOnRecvPacketBefore interface {
	OnRecvPacketBeforeHook(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress)
}
type IBCAppHooksOnRecvPacketAfter interface {
	OnRecvPacketAfterHook(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress, ack ibcexported.Acknowledgement)
}

// SendPacket Hooks
type IBCAppHooksSendPacketOverride interface {
	SendPacketOverride(im IBCMiddleware, ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI) error
}
type IBCAppHooksSendPacketBefore interface {
	SendPacketBeforeHook(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI)
}
type IBCAppHooksSendPacketAfter interface {
	SendPacketAfterHook(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, err error)
}

// WriteAcknowledgement Hooks
type IBCAppHooksWriteAcknowledgementOverride interface {
	WriteAcknowledgementOverride(im IBCMiddleware, ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error
}
type IBCAppHooksWriteAcknowledgementBefore interface {
	WriteAcknowledgementBeforeHook(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, ack ibcexported.Acknowledgement)
}
type IBCAppHooksWriteAcknowledgementAfter interface {
	WriteAcknowledgementAfterHook(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, ack ibcexported.Acknowledgement, err error)
}

// GetAppVersion Hooks
type IBCAppHooksGetAppVersionOverride interface {
	GetAppVersionOverride(im IBCMiddleware, ctx sdk.Context, portID, channelID string) (string, bool)
}
type IBCAppHooksGetAppVersionBefore interface {
	GetAppVersionBeforeHook(ctx sdk.Context, portID, channelID string)
}
type IBCAppHooksGetAppVersionAfter interface {
	GetAppVersionAfterHook(ctx sdk.Context, portID, channelID string, result string, success bool)
}
