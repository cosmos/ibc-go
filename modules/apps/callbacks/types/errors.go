package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrNotCallbackPacketData = errorsmod.Register(ModuleName, 2, "packet is not a CallbackPacketData")
	ErrCallbackOutOfGas      = errorsmod.Register(ModuleName, 3, "callback out of gas")
)
