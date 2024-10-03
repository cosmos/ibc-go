package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC channel sentinel errors
var (
	ErrInvalidPacket  = errorsmod.Register(SubModuleName, 1, "invalid packet")
	ErrInvalidPayload = errorsmod.Register(SubModuleName, 2, "invalid payload")
)
