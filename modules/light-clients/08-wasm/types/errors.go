package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrInvalidData              = sdkerrors.Register(ModuleName, 1, "invalid data")
	ErrInvalidCodeId            = sdkerrors.Register(ModuleName, 2, "invalid code ID")
	ErrInvalidHeader            = sdkerrors.Register(ModuleName, 3, "invalid header")
	ErrUnableToUnmarshalPayload = sdkerrors.Register(ModuleName, 4, "unable to unmarshal wasm contract return value")
	ErrUnableToInit             = sdkerrors.Register(ModuleName, 5, "unable to initialize wasm contract")
	ErrUnableToCall             = sdkerrors.Register(ModuleName, 6, "unable to call wasm contract")
	ErrUnableToQuery            = sdkerrors.Register(ModuleName, 7, "unable to query wasm contract")
	ErrUnableToMarshalPayload   = sdkerrors.Register(ModuleName, 8, "unable to marshal wasm contract payload")
	// Wasm specific
	ErrWasmEmptyCode      = sdkerrors.Register(ModuleName, 9, "empty wasm code")
	ErrWasmEmptyCodeHash  = sdkerrors.Register(ModuleName, 10, "empty wasm code hash")
	ErrWasmCodeTooLarge   = sdkerrors.Register(ModuleName, 11, "wasm code too large")
	ErrWasmCodeExists     = sdkerrors.Register(ModuleName, 12, "wasm code already exists")
	ErrWasmCodeValidation = sdkerrors.Register(ModuleName, 13, "unable to validate wasm code")
	ErrWasmInvalidCode    = sdkerrors.Register(ModuleName, 14, "invalid wasm code")
	ErrWasmInvalidCodeID  = sdkerrors.Register(ModuleName, 15, "invalid wasm code id")
	ErrWasmCodeIDNotFound = sdkerrors.Register(ModuleName, 16, "wasm code id not found")
	ErrInvalid            = sdkerrors.Register(ModuleName, 17, "invalid")
	ErrCreateFailed       = sdkerrors.Register(ModuleName, 18, "create wasm contract failed")
)
