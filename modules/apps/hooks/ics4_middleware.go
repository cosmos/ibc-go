package ibc_hooks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	porttypes "github.com/cosmos/ibc-go/v5/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v5/modules/core/exported"
)

var (
	_ porttypes.ICS4Wrapper = &ICS4Middleware{}
)

type ICS4Middleware struct {
	channel porttypes.ICS4Wrapper
}

func NewICS4Middleware(channel porttypes.ICS4Wrapper) ICS4Middleware {
	return ICS4Middleware{
		channel: channel,
	}
}

func (i ICS4Middleware) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
	return i.channel.SendPacket(ctx, channelCap, packet)
}

func (i ICS4Middleware) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	return i.channel.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

func (i ICS4Middleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return i.channel.GetAppVersion(ctx, portID, channelID)
}
