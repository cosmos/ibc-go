package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrWasmEmptyCode      = sdkerrors.Register(SubModuleName, 2, "empty wasm code")
	ErrWasmCodeExists     = sdkerrors.Register(SubModuleName, 3, "wasm code already exists")
	ErrWasmCodeValidation = sdkerrors.Register(SubModuleName, 4, "unable to validate wasm code")
	ErrWasmInvalidCode    = sdkerrors.Register(SubModuleName, 5, "invalid wasm code")
	ErrWasmInvalidCodeID  = sdkerrors.Register(SubModuleName, 6, "invalid wasm code id")
	ErrWasmCodeIDNotFound = sdkerrors.Register(SubModuleName, 7, "wasm code id not found")
)
