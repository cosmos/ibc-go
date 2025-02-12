package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidCounterparty  = errorsmod.Register(SubModuleName, 34, "invalid counterparty")
	ErrCounterpartyNotFound = errorsmod.Register(SubModuleName, 35, "counterparty not found")
)
