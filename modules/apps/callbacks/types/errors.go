package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrCannotUnmarshalPacketData = errorsmod.Register(ModuleName, 2, "cannot unmarshal packet data")
	ErrNotPacketDataProvider     = errorsmod.Register(ModuleName, 3, "packet is not a PacketDataProvider")
	ErrCallbackKeyNotFound       = errorsmod.Register(ModuleName, 4, "callback key not found in packet data")
	ErrCallbackAddressNotFound   = errorsmod.Register(ModuleName, 5, "callback address not found in packet data")
	ErrCallbackOutOfGas          = errorsmod.Register(ModuleName, 6, "callback out of gas")
	ErrCallbackPanic             = errorsmod.Register(ModuleName, 7, "callback panic")
)
