package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CCV sentinel errors
var (
	ErrInvalidPacketData        = sdkerrors.Register(ModuleName, 2, "invalid CCV packet data")
	ErrInvalidPacketTimeout     = sdkerrors.Register(ModuleName, 3, "invalid packet timeout")
	ErrInvalidVersion           = sdkerrors.Register(ModuleName, 4, "invalid CCV version")
	ErrInvalidChannelFlow       = sdkerrors.Register(ModuleName, 5, "invalid message sent to channel end")
	ErrInvalidChildChain        = sdkerrors.Register(ModuleName, 6, "invalid child chain")
	ErrInvalidParentChain       = sdkerrors.Register(ModuleName, 7, "invalid parent chain")
	ErrInvalidStatus            = sdkerrors.Register(ModuleName, 8, "invalid channel status")
	ErrInvalidGenesis           = sdkerrors.Register(ModuleName, 9, "invalid genesis state")
	ErrDuplicateChannel         = sdkerrors.Register(ModuleName, 10, "CCV channel already exists")
	ErrInvalidUnbondingSequence = sdkerrors.Register(ModuleName, 11, "invalid unbonding sequence")
	ErrInvalidUnbondingTime     = sdkerrors.Register(ModuleName, 12, "child chain has invalid unbonding time")
	ErrInvalidChildState        = sdkerrors.Register(ModuleName, 13, "parent chain has invalid state for child chain")
	ErrInvalidChildClient       = sdkerrors.Register(ModuleName, 14, "ccv channel is not built on correct client")
	ErrInvalidProposal          = sdkerrors.Register(ModuleName, 15, "invalid create child chain proposal")
)
