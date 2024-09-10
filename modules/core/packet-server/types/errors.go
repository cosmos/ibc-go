package types

import (
	errorsmod "cosmossdk.io/errors"
)

var ErrInvalidCounterparty = errorsmod.Register(SubModuleName, 1, "invalid counterparty")
