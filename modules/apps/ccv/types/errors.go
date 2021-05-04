package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CCV sentinel errors
var (
	ErrInvalidPacketTimeout = sdkerrors.Register(ModuleName, 2, "invalid packet timeout")
	ErrInvalidVersion       = sdkerrors.Register(ModuleName, 3, "invalid CCV version")
	ErrInvalidChannelFlow   = sdkerrors.Register(ModuleName, 4, "invalid message sent to channel end")
	ErrInvalidChildChain    = sdkerrors.Register(ModuleName, 5, "invalid child chain")
	ErrInvalidStatus        = sdkerrors.Register(ModuleName, 6, "invalid channel status")
)
