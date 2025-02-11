package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidPacket            = errorsmod.Register(SubModuleName, 2, "invalid packet")
	ErrInvalidPayload           = errorsmod.Register(SubModuleName, 3, "invalid payload")
	ErrSequenceSendNotFound     = errorsmod.Register(SubModuleName, 4, "sequence send not found")
	ErrInvalidAcknowledgement   = errorsmod.Register(SubModuleName, 5, "invalid acknowledgement")
	ErrPacketCommitmentNotFound = errorsmod.Register(SubModuleName, 6, "packet commitment not found")
	ErrAcknowledgementNotFound  = errorsmod.Register(SubModuleName, 7, "packet acknowledgement not found")
	ErrInvalidTimeout           = errorsmod.Register(SubModuleName, 8, "invalid packet timeout")
	ErrTimeoutElapsed           = errorsmod.Register(SubModuleName, 9, "timeout elapsed")
	ErrTimeoutNotReached        = errorsmod.Register(SubModuleName, 10, "timeout not reached")
	ErrAcknowledgementExists    = errorsmod.Register(SubModuleName, 11, "acknowledgement for packet already exists")
	ErrNoOpMsg                  = errorsmod.Register(SubModuleName, 12, "message is redundant, no-op will be performed")
)
