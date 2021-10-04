package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// 29-fee sentinel errors
var (
	ErrInvalidVersion    = sdkerrors.Register(ModuleName, 1, "invalid ICS29 middleware version")
	ErrRefundAccNotFound = sdkerrors.Register(ModuleName, 2, "No account found for given refund address")
	ErrBalanceNotFound   = sdkerrors.Register(ModuleName, 3, "Balance not found for given account address")
)
