package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC packet server sentinel errors
var (
	ErrInvalidCounterparty = errorsmod.Register(SubModuleName, 1, "invalid counterparty")
)
