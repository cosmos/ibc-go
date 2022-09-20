package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	SubModuleName         = "wasm_manager"
	ErrWasmEmptyCode      = sdkerrors.Register(SubModuleName, 2, "empty wasm code")
	ErrWasmEmptyCodeHash  = sdkerrors.Register(SubModuleName, 3, "empty wasm code hash")
	ErrWasmCodeExists     = sdkerrors.Register(SubModuleName, 4, "wasm code already exists")
	ErrWasmCodeValidation = sdkerrors.Register(SubModuleName, 5, "unable to validate wasm code")
	ErrWasmInvalidCode    = sdkerrors.Register(SubModuleName, 6, "invalid wasm code")
	ErrWasmInvalidCodeID  = sdkerrors.Register(SubModuleName, 7, "invalid wasm code id")
	ErrWasmCodeIDNotFound = sdkerrors.Register(SubModuleName, 8, "wasm code id not found")
)
