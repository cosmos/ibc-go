package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrUnknownDataType    = sdkerrors.Register(ModuleName, 1, "unknown data type")
	ErrInvalidChannelFlow = sdkerrors.Register(ModuleName, 2, "invalid message sent to channel end")
	ErrInvalidHostPort    = sdkerrors.Register(ModuleName, 3, "invalid host port")
	ErrHostDisabled       = sdkerrors.Register(ModuleName, 4, "host is disabled")
	ErrInvalidVersion     = sdkerrors.Register(ModuleName, 5, "invalid version")
)
