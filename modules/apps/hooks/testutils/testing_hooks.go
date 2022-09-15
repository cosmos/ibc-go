package testutils

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	ibchooks "github.com/cosmos/ibc-go/v5/modules/apps/hooks"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v5/modules/core/exported"
)

var _ ibchooks.IBCAppHooks = TestRecvOverrides{}
var _ ibchooks.IBCAppHooks = TestRecvBeforeAfterHooks{}

type Status struct {
	OverrideRan bool
	BeforeRan   bool
	AfterRan    bool
}

// Recv
type TestRecvOverrides struct{ Status *Status }

func (t TestRecvOverrides) OnRecvPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) ibcexported.Acknowledgement {
	t.Status.OverrideRan = true
	ack := im.App.OnRecvPacket(ctx, packet, relayer)
	return ack
}

type TestRecvBeforeAfterHooks struct{ Status *Status }

func (t TestRecvBeforeAfterHooks) OnRecvPacketBeforeHook(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) {
	t.Status.BeforeRan = true
}
func (t TestRecvBeforeAfterHooks) OnRecvPacketAfterHook(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress, ack ibcexported.Acknowledgement) {
	t.Status.AfterRan = true
}

// Send
type TestSendOverrides struct{ Status *Status }

func (t TestSendOverrides) SendPacketOverride(im ibchooks.IBCMiddleware, ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
	t.Status.OverrideRan = true
	err := im.SendPacket(ctx, chanCap, packet)
	return err
}

type TestSendBeforeAfterHooks struct{ Status *Status }

func (t TestSendBeforeAfterHooks) SendPacketBeforeHook(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI) {
	t.Status.BeforeRan = true
}
func (t TestSendBeforeAfterHooks) SendPacketAfterHook(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, err error) {
	t.Status.AfterRan = true
}
