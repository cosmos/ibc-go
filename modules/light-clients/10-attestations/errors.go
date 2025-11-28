package attestations

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidAttestationProof = errorsmod.Register(ModuleName, 2, "invalid attestation proof")
	ErrInvalidSignature        = errorsmod.Register(ModuleName, 3, "invalid signature")
	ErrInvalidQuorum           = errorsmod.Register(ModuleName, 4, "quorum threshold not met")
	ErrDuplicateSigner         = errorsmod.Register(ModuleName, 5, "duplicate signer")
	ErrUnknownSigner           = errorsmod.Register(ModuleName, 6, "unknown signer")
	ErrInvalidAttestationData  = errorsmod.Register(ModuleName, 7, "invalid attestation data")
	ErrInvalidHeight           = errorsmod.Register(ModuleName, 8, "invalid height")
	ErrInvalidTimestamp        = errorsmod.Register(ModuleName, 9, "invalid timestamp")
	ErrClientFrozen            = errorsmod.Register(ModuleName, 10, "client is frozen")
	ErrNotMember               = errorsmod.Register(ModuleName, 11, "value not found in attested packet commitments")
	ErrInvalidPath             = errorsmod.Register(ModuleName, 12, "invalid path")
	ErrProcessedTimeNotFound   = errorsmod.Register(ModuleName, 13, "processed time not found")
	ErrProcessedHeightNotFound = errorsmod.Register(ModuleName, 14, "processed height not found")
	ErrDelayPeriodNotPassed    = errorsmod.Register(ModuleName, 15, "delay period has not been reached")
)
