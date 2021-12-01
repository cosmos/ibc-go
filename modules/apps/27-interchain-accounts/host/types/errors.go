package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ICA Host sentinel errors
var (
	ErrHostSubModuleDisabled = sdkerrors.Register(ModuleName, 2, "host submodule is disabled")
)
