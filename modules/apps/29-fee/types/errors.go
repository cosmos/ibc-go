package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// 29-fee sentinel errors
var (
	ErrInvalidVersion    = sdkerrors.Register(ModuleName, 1, "invalid ICS29 middleware version")
	ErrRefundAccNotFound = sdkerrors.Register(ModuleName, 2, "no account found for given refund address")
	ErrBalanceNotFound   = sdkerrors.Register(ModuleName, 3, "balance not found for given account address")
	ErrFeeNotFound       = sdkerrors.Register(ModuleName, 4, "there is no fee escrowed for the given packetId")
	ErrPayingFee         = sdkerrors.Register(ModuleName, 5, "error while paying fee")
	ErrRefundingFee      = sdkerrors.Register(ModuleName, 6, "error while refunding fee")
	ErrFeeEmpty          = sdkerrors.Register(ModuleName, 7, "fee struct cannot be empty")
	ErrRelayersNotNil    = sdkerrors.Register(ModuleName, 8, "relayers must be nil. This feature is not supported")
)
