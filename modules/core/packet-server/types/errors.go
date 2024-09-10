package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidCounterparty  = errorsmod.Register(SubModuleName, 1, "invalid counterparty")
	ErrCounterpartyNotFound = errorsmod.Register(SubModuleName, 2, "counterparty not found")
)
