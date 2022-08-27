package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// 31-ibc-query sentinel errors
var (
	ErrInvalidVersion          = sdkerrors.Register(ModuleName, 2, "invalid 31-IBC-query version")
	ErrCrossChainQueryNotFound = sdkerrors.Register(ModuleName, 3, "no query found for given query id")
	ErrQuerytTimeout             = sdkerrors.Register(ModuleName, 4, "query timeout")
	ErrInvalidTimeoutHeight      = sdkerrors.Register(ModuleName, 5, "invalid timeout height")
)
