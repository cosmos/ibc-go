package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidChannel       = errorsmod.Register(SubModuleName, 1, "invalid channel")
	ErrChannelNotFound      = errorsmod.Register(SubModuleName, 2, "channel not found")
	ErrInvalidPacket        = errorsmod.Register(SubModuleName, 3, "invalid packet")
	ErrInvalidPayload       = errorsmod.Register(SubModuleName, 4, "invalid payload")
	ErrSequenceSendNotFound = errorsmod.Register(SubModuleName, 5, "sequence send not found")
)
