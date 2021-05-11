package types

import sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

var (
	ErrInvalidHeader          = sdkerrors.Register(SubModuleName, 1, "invalid header")
	ErrUnableToUnmarshalPayload = sdkerrors.Register(SubModuleName, 2, "unable to unmarshal wasm contract return value")
	ErrUnableToInit = sdkerrors.Register(SubModuleName, 3, "unable to initialize wasm contract")
	ErrUnableToCall = sdkerrors.Register(SubModuleName, 4, "unable to call wasm contract")
	ErrUnableToQuery = sdkerrors.Register(SubModuleName, 5, "unable to query wasm contract")
	ErrUnableToMarshalPayload = sdkerrors.Register(SubModuleName, 6, "unable to marshal wasm contract payload")
)