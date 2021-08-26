package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/transfer/types"
)

var _ types.TransferHooks = Keeper{}

func (k Keeper) AfterSendTransfer(
	ctx sdk.Context,
	sourcePort, sourceChannel string,
	token sdk.Coin,
	sender sdk.AccAddress,
	receiver string,
	isSource bool) {
	if k.hooks != nil {
		k.hooks.AfterSendTransfer(ctx, sourcePort, sourceChannel, token, sender, receiver, isSource)
	}
}

func (k Keeper) AfterRecvTransfer(
	ctx sdk.Context,
	destPort, destChannel string,
	token sdk.Coin,
	receiver string,
	isSource bool) {
	if k.hooks != nil {
		k.hooks.AfterRecvTransfer(ctx, destPort, destChannel, token, receiver, isSource)
	}
}

func (k Keeper) AfterRefundTransfer(
	ctx sdk.Context,
	sourcePort, sourceChannel string,
	token sdk.Coin,
	sender string,
	isSource bool) {
	if k.hooks != nil {
		k.hooks.AfterRefundTransfer(ctx, sourcePort, sourceChannel, token, sender, isSource)
	}
}

func (k *Keeper) SetHooks(sh types.TransferHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set hooks twice")
	}

	k.hooks = sh

	return k
}
