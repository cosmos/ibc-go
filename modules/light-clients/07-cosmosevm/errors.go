package cosmosevm

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC tendermint client sentinel errors
var (
	ErrInvalidClientType = errorsmod.Register(ModuleName, 2, "invalid chain-type")
	ErrTendermintClientNotFound = errorsmod.Register(ModuleName, 3, "tendermint client not found")
	ErrInvalidTendermintClientState = errorsmod.Register(ModuleName, 4, "invalid tendermint client state")
	ErrUpdatesNotAllowed = errorsmod.Register(ModuleName, 5, "updates to cosmosevm client state are not allowed")
)
