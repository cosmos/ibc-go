package cosmosevm

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC tendermint client sentinel errors
var (
	ErrInvalidClientType = errorsmod.Register(ModuleName, 2, "invalid chain-type")
)
