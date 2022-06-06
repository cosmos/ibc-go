package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrUnknownDataType       = sdkerrors.Register(ModuleName, 1, "unknown data type")
	ErrInvalidChannelFlow    = sdkerrors.Register(ModuleName, 2, "invalid message sent to channel end")
	ErrInvalidControllerPort = sdkerrors.Register(ModuleName, 3, "invalid controller port")
	ErrInvalidHostPort       = sdkerrors.Register(ModuleName, 4, "invalid host port")
	ErrHostDisabled          = sdkerrors.Register(ModuleName, 5, "host is disabled")
)
