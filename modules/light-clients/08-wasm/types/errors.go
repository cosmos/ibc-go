package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalid              = errorsmod.Register(ModuleName, 1, "invalid")
	ErrInvalidData          = errorsmod.Register(ModuleName, 2, "invalid data")
	ErrInvalidCodeID        = errorsmod.Register(ModuleName, 3, "invalid code ID")
	ErrInvalidClientMessage = errorsmod.Register(ModuleName, 4, "invalid client message")
	// Wasm specific
	ErrWasmEmptyCode      = errorsmod.Register(ModuleName, 5, "empty wasm code")
	ErrWasmCodeTooLarge   = errorsmod.Register(ModuleName, 6, "wasm code too large")
	ErrWasmCodeExists     = errorsmod.Register(ModuleName, 7, "wasm code already exists")
	ErrWasmCodeIDNotFound = errorsmod.Register(ModuleName, 8, "wasm code id not found")
)
