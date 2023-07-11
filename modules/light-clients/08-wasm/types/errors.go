package types

var (
	ErrInvalid       = sdkerrors.Register(ModuleName, 1, "invalid")
	ErrInvalidData   = sdkerrors.Register(ModuleName, 2, "invalid data")
	ErrInvalidCodeID = sdkerrors.Register(ModuleName, 3, "invalid code ID")
	// Wasm specific
	ErrWasmEmptyCode      = sdkerrors.Register(ModuleName, 4, "empty wasm code")
	ErrWasmCodeTooLarge   = sdkerrors.Register(ModuleName, 5, "wasm code too large")
	ErrWasmCodeExists     = sdkerrors.Register(ModuleName, 6, "wasm code already exists")
	ErrWasmCodeIDNotFound = sdkerrors.Register(ModuleName, 7, "wasm code id not found")
)
