package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TransferHooks interface {
	AfterTransferEnd(ctx sdk.Context, data FungibleTokenPacketData, base_denom string)
}

var _ TransferHooks = MultiTransferHooks{}

type MultiTransferHooks []TransferHooks

func NewMultiTransferHooks(hooks ...TransferHooks) MultiTransferHooks {
	return hooks
}

func (h MultiTransferHooks) AfterTransferEnd(ctx sdk.Context, data FungibleTokenPacketData, base_denom string) {
	for i := range h {
		h[i].AfterTransferEnd(ctx, data, base_denom)
	}
}
