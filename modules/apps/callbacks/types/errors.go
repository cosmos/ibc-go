package types

import (
	errorsmod "cosmossdk.io/errors"
)

// IBC client sentinel errors
var (
	ErrNotCallbackPacketData = errorsmod.Register(ModuleName, 2, "packet is not a CallbackPacketData")
)