package types

import (
	errorsmod "cosmossdk.io/errors"
)

// TokenFactory sentinel errors
var (
	ErrInvalidRequest           = errorsmod.Register(ModuleName, 1, "invalid request")
	ErrDenomExists              = errorsmod.Register(ModuleName, 2, "attempting to create a denom that already exists (has bank metadata)")
	ErrUnauthorized             = errorsmod.Register(ModuleName, 3, "unauthorized account")
	ErrInvalidDenom             = errorsmod.Register(ModuleName, 4, "invalid denom")
	ErrInvalidCreator           = errorsmod.Register(ModuleName, 5, "invalid creator")
	ErrInvalidAddress           = errorsmod.Register(ModuleName, 6, "invalid address")
	ErrDenomNotFound            = errorsmod.Register(ModuleName, 8, "denom not found")
	ErrCreatorNotFound          = errorsmod.Register(ModuleName, 9, "creator not found")
	ErrInvalidGenesis           = errorsmod.Register(ModuleName, 10, "invalid genesis")
	ErrDenomTooLong             = errorsmod.Register(ModuleName, 11, "denom too long")
	ErrInvalidAuthorityMetadata = errorsmod.Register(ModuleName, 12, "invalid authority metadata")
	ErrInvalidAmount            = errorsmod.Register(ModuleName, 13, "invalid amount")
	ErrAdminRenounced           = errorsmod.Register(ModuleName, 14, "admin has been renounced")
)
