package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidChannel  = errorsmod.Register(SubModuleName, 1, "invalid counterparty")
	ErrChannelNotFound = errorsmod.Register(SubModuleName, 2, "counterparty not found")
)
