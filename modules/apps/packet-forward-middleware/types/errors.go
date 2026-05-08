package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrMetadataKeyNotFound    = errorsmod.Register(ModuleName, 1, "metadata key not found in packet data")
	ErrInvalidForwardMetadata = errorsmod.Register(ModuleName, 2, "invalid forward metadata")
	ErrForwardTransferFailed  = errorsmod.Register(ModuleName, 3, "failed to forward transfer packet")
)
