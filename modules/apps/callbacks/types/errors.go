package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrNotCallbackPacketData = errorsmod.Register(ModuleName, 2, "packet is not a CallbackPacketData")
	ErrCallbackPanic         = errorsmod.Register(ModuleName, 3, "callback execution panicked")
)
