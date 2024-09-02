package mock

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v9/modules/core/05-port/types"
)

var (
	_ porttypes.IBCModuleV2 = (*IBCModuleV2)(nil)
)

// IBCModuleV2 implements the IBCModuleV2 interface for testing v2 of the IBC module.
type IBCModuleV2 struct {
	IBCApp *IBCAppV2 // base application of an IBC middleware stack
}

func (im IBCModuleV2) OnSendPacketV2(ctx sdk.Context, portID string, channelID string, sequence uint64, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, payload channeltypes.Payload, signer sdk.AccAddress) error {
	if im.IBCApp.OnSendPacketV2 != nil {
		return im.IBCApp.OnSendPacketV2(ctx, portID, channelID, sequence, timeoutHeight, timeoutTimestamp, payload, signer)
	}

	return nil
}

func (im IBCModuleV2) OnRecvPacketV2(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) channeltypes.RecvPacketResult {
	if im.IBCApp.OnRecvPacketV2 != nil {
		return im.IBCApp.OnRecvPacketV2(ctx, packet, payload, relayer)
	}

	return channeltypes.RecvPacketResult{
		Status:          channeltypes.PacketStatus_Success,
		Acknowledgement: channeltypes.NewResultAcknowledgement([]byte("success")).Acknowledgement(),
	}
}

func (im IBCModuleV2) OnAcknowledgementPacketV2(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, recvPacketResult channeltypes.RecvPacketResult, relayer sdk.AccAddress) error {
	if im.IBCApp.OnAcknowledgementPacketV2 != nil {
		return im.IBCApp.OnAcknowledgementPacketV2(ctx, packet, payload, recvPacketResult, relayer)
	}

	return nil
}

func (im IBCModuleV2) OnTimeoutPacketV2(ctx sdk.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, relayer sdk.AccAddress) error {
	if im.IBCApp.OnTimeoutPacketV2 != nil {
		return im.IBCApp.OnTimeoutPacketV2(ctx, packet, payload, relayer)
	}

	return nil
}

// NewIBCModuleV2 creates a new IBCModule given the underlying mock IBC application and scopedKeeper.
func NewIBCModuleV2(app *IBCAppV2) IBCModuleV2 {
	return IBCModuleV2{
		IBCApp: app,
	}
}
