package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidCodec            = errorsmod.Register(ModuleName, 1, "codec is not supported")
	ErrInvalidMemo             = errorsmod.Register(ModuleName, 2, "invalid memo")
	ErrInvalidSalt             = errorsmod.Register(ModuleName, 3, "invalid salt")
	ErrInvalidPayload          = errorsmod.Register(ModuleName, 4, "invalid payload")
	ErrInvalidTimeoutTimestamp = errorsmod.Register(ModuleName, 5, "invalid timeout timestamp")
	ErrInvalidEncoding         = errorsmod.Register(ModuleName, 6, "invalid encoding")
	ErrAbiDecoding             = errorsmod.Register(ModuleName, 7, "abi decoding error")
	ErrAbiEncoding             = errorsmod.Register(ModuleName, 8, "abi encoding error")
	ErrAccountAlreadyExists    = errorsmod.Register(ModuleName, 9, "account already exists")
)
