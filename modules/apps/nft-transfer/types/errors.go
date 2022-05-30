package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// IBC transfer sentinel errors
var (
	ErrInvalidPacketTimeout = sdkerrors.Register(ModuleName, 2, "invalid packet timeout")
	ErrInvalidVersion       = sdkerrors.Register(ModuleName, 3, "invalid ICS721 version")
	ErrMaxTransferChannels  = sdkerrors.Register(ModuleName, 4, "max nft-transfer channels")
	ErrInvalidClassID       = sdkerrors.Register(ModuleName, 5, "invalid class id")
	ErrInvalidTokenID       = sdkerrors.Register(ModuleName, 5, "invalid token id")
	ErrInvalidPacket        = sdkerrors.Register(ModuleName, 5, "invalid non-fungible token packet")
)
