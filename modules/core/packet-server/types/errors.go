package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC packet server sentinel errors
var (
	ErrInvalidCounterparty = errorsmod.Register(SubModuleName, 1, "invalid counterparty")
	ErrInvalidPacketPath   = errorsmod.Register(SubModuleName, 2, "invalid packet path")
)
