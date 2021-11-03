package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// 29-fee sentinel errors
var (
	ErrInvalidVersion    = sdkerrors.Register(ModuleName, 2, "invalid ICS29 middleware version")
	ErrRefundAccNotFound = sdkerrors.Register(ModuleName, 3, "no account found for given refund address")
	ErrBalanceNotFound   = sdkerrors.Register(ModuleName, 4, "balance not found for given account address")
	ErrFeeNotFound       = sdkerrors.Register(ModuleName, 5, "there is no fee escrowed for the given packetID")
	ErrRelayersNotNil    = sdkerrors.Register(ModuleName, 6, "relayers must be nil. This feature is not supported")
)
