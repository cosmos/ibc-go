package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidChannel           = errorsmod.Register(SubModuleName, 2, "invalid channel")
	ErrChannelNotFound          = errorsmod.Register(SubModuleName, 3, "channel not found")
	ErrInvalidPacket            = errorsmod.Register(SubModuleName, 4, "invalid packet")
	ErrInvalidPayload           = errorsmod.Register(SubModuleName, 5, "invalid payload")
	ErrSequenceSendNotFound     = errorsmod.Register(SubModuleName, 6, "sequence send not found")
	ErrInvalidAcknowledgement   = errorsmod.Register(SubModuleName, 8, "invalid acknowledgement")
	ErrPacketCommitmentNotFound = errorsmod.Register(SubModuleName, 9, "packet commitment not found")
)
