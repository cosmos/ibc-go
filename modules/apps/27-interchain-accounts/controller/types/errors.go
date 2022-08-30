package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ICA Controller sentinel errors
var (
	ErrControllerSubModuleDisabled = sdkerrors.Register(SubModuleName, 1, "controller submodule is disabled")
)
