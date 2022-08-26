package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ICA Controller sentinel errors
var (
	ErrControllerSubModuleDisabled = sdkerrors.Register(SubModuleName, 1, "controller submodule is disabled")
	ErrInvalidTimeout              = sdkerrors.Register(SubModuleName, 2, "timeout height and timestamp cannot both be zero")
	ErrEmptyMsgs                   = sdkerrors.Register(SubModuleName, 3, "interchain accounts data packets array cannot be empty")
)
