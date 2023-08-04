package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrNotPacketDataProvider = errorsmod.Register(ModuleName, 2, "packet is not a PacketDataProvider")
	ErrCallbackKeyNotFound   = errorsmod.Register(ModuleName, 3, "callback key not found in packet data")
)
