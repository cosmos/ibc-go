package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrUnknownPacketData           = sdkerrors.Register(ModuleName, 1, "Unknown packet data")
	ErrAccountAlreadyExist         = sdkerrors.Register(ModuleName, 2, "Account already exist")
	ErrPortAlreadyBound            = sdkerrors.Register(ModuleName, 3, "Port is already bound for address")
	ErrUnsupportedChain            = sdkerrors.Register(ModuleName, 4, "Unsupported chain")
	ErrInvalidOutgoingData         = sdkerrors.Register(ModuleName, 5, "Invalid outgoing data")
	ErrInvalidRoute                = sdkerrors.Register(ModuleName, 6, "Invalid route")
	ErrInterchainAccountNotFound   = sdkerrors.Register(ModuleName, 7, "Interchain account not found")
	ErrInterchainAccountAlreadySet = sdkerrors.Register(ModuleName, 8, "Interchain Account is already set")
	ErrActiveChannelNotFound       = sdkerrors.Register(ModuleName, 9, "No active channel for this owner")
)
