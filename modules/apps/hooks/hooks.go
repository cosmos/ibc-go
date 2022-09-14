package ibc_hooks

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v5/modules/core/exported"
)

type TransferHooks interface {
}

type TransferHooksOnRecvPacketOverride interface {
	OnRecvPacketOverride(IBCMiddleware, sdk.Context, channeltypes.Packet, sdk.AccAddress) ibcexported.Acknowledgement
}
type TransferHooksOnRecvPacketBefore interface {
	OnRecvPacketBeforeHook(sdk.Context, channeltypes.Packet, sdk.AccAddress)
}
type TransferHooksOnRecvPacketAfter interface {
	OnRecvPacketAfterHook(sdk.Context, channeltypes.Packet, sdk.AccAddress)
}
