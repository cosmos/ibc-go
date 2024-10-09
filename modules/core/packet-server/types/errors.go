package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidChannel  = errorsmod.Register(SubModuleName, 1, "invalid channel")
	ErrChannelNotFound = errorsmod.Register(SubModuleName, 2, "channel not found")
)
