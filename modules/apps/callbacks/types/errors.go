package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrNotPacketDataProvider   = errorsmod.Register(ModuleName, 2, "packet is not a PacketDataProvider")
	ErrCallbackMemoKeyNotFound = errorsmod.Register(ModuleName, 3, "callback memo key not found")
)
