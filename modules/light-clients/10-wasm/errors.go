package wasm

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrInvalidHeader            = sdkerrors.Register(SubModuleName, 1, "invalid header")
	ErrUnableToUnmarshalPayload = sdkerrors.Register(SubModuleName, 2, "unable to unmarshal wasm contract return value")
	ErrUnableToInit             = sdkerrors.Register(SubModuleName, 3, "unable to initialize wasm contract")
	ErrUnableToCall             = sdkerrors.Register(SubModuleName, 4, "unable to call wasm contract")
	ErrUnableToQuery            = sdkerrors.Register(SubModuleName, 5, "unable to query wasm contract")
	ErrUnableToMarshalPayload   = sdkerrors.Register(SubModuleName, 6, "unable to marshal wasm contract payload")
	// Wasm specific
	ErrWasmEmptyCode      = sdkerrors.Register(SubModuleName, 7, "empty wasm code")
	ErrWasmEmptyCodeHash  = sdkerrors.Register(SubModuleName, 8, "empty wasm code hash")
	ErrWasmCodeExists     = sdkerrors.Register(SubModuleName, 9, "wasm code already exists")
	ErrWasmCodeValidation = sdkerrors.Register(SubModuleName, 10, "unable to validate wasm code")
	ErrWasmInvalidCode    = sdkerrors.Register(SubModuleName, 11, "invalid wasm code")
	ErrWasmInvalidCodeID  = sdkerrors.Register(SubModuleName, 12, "invalid wasm code id")
	ErrWasmCodeIDNotFound = sdkerrors.Register(SubModuleName, 13, "wasm code id not found")
)
