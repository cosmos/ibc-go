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
	ErrInvalidAcknowledgement   = errorsmod.Register(SubModuleName, 7, "invalid acknowledgement")
	ErrPacketCommitmentNotFound = errorsmod.Register(SubModuleName, 8, "packet commitment not found")
	ErrAcknowledgementNotFound  = errorsmod.Register(SubModuleName, 9, "packet acknowledgement not found")
	ErrTimeoutElapsed           = errorsmod.Register(SubModuleName, 10, "timeout elapsed")
	ErrInvalidChannelIdentifier = errorsmod.Register(SubModuleName, 11, "invalid channel identifier")
	ErrAcknowledgementExists    = errorsmod.Register(SubModuleName, 12, "acknowledgement for packet already exists")
	ErrTimeoutNotReached        = errorsmod.Register(SubModuleName, 13, "timeout not reached")
	// Perform a no-op on the current Msg
	ErrNoOpMsg        = errorsmod.Register(SubModuleName, 14, "message is redundant, no-op will be performed")
	ErrInvalidTimeout = errorsmod.Register(SubModuleName, 15, "invalid packet timeout")
)
