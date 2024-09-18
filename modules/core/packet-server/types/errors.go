package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC packet server sentinel errors
var (
	ErrInvalidCounterparty  = errorsmod.Register(SubModuleName, 1, "invalid counterparty")
	ErrCounterpartyNotFound = errorsmod.Register(SubModuleName, 2, "counterparty not found")
	ErrInvalidPacketPath    = errorsmod.Register(SubModuleName, 3, "invalid packet path")
)
