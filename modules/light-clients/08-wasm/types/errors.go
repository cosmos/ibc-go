package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrInvalidHeader            = sdkerrors.Register(ModuleName, 1, "invalid header")
	ErrUnableToUnmarshalPayload = sdkerrors.Register(ModuleName, 2, "unable to unmarshal wasm contract return value")
	ErrUnableToInit             = sdkerrors.Register(ModuleName, 3, "unable to initialize wasm contract")
	ErrUnableToCall             = sdkerrors.Register(ModuleName, 4, "unable to call wasm contract")
	ErrUnableToQuery            = sdkerrors.Register(ModuleName, 5, "unable to query wasm contract")
	ErrUnableToMarshalPayload   = sdkerrors.Register(ModuleName, 6, "unable to marshal wasm contract payload")
	// Wasm specific
	ErrWasmEmptyCode      = sdkerrors.Register(ModuleName, 7, "empty wasm code")
	ErrWasmEmptyCodeHash  = sdkerrors.Register(ModuleName, 8, "empty wasm code hash")
	ErrWasmCodeTooLarge   = sdkerrors.Register(ModuleName, 9, "wasm code too large")
	ErrWasmCodeExists     = sdkerrors.Register(ModuleName, 10, "wasm code already exists")
	ErrWasmCodeValidation = sdkerrors.Register(ModuleName, 11, "unable to validate wasm code")
	ErrWasmInvalidCode    = sdkerrors.Register(ModuleName, 12, "invalid wasm code")
	ErrWasmInvalidCodeID  = sdkerrors.Register(ModuleName, 13, "invalid wasm code id")
	ErrWasmCodeIDNotFound = sdkerrors.Register(ModuleName, 14, "wasm code id not found")
)
