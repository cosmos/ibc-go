package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrEmptyClientType = sdkerrors.Register(SubModuleName, 2, "empty client type")
	ErrEmptyWASMCode = sdkerrors.Register(SubModuleName, 3, "empty wasm code")
)