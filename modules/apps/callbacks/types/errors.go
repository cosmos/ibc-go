package types

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrNotAdditionalPacketDataProvider = errorsmod.Register(ModuleName, 2, "packet is not a AdditionalPacketDataProvider")
	ErrCallbackMemoKeyNotFound         = errorsmod.Register(ModuleName, 3, "callback memo key not found")
)
