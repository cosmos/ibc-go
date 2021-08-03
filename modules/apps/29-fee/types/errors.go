package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// 29-fee sentinel errors
var (
	ErrInvalidVersion = sdkerrors.Register(ModuleName, 2, "invalid ICS29 middleware version")
)
