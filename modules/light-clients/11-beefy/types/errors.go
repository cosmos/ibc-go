package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	SubModuleName = "beefy-client"
)

// IBC beefy client errors
var (
	ErrInvalidChainID      = sdkerrors.Register(SubModuleName, 0, "invalid chain-id")
	ErrInvalidRootHash     = sdkerrors.Register(SubModuleName, 1, "invalid root hash")
	ErrInvalidHeaderHeight = sdkerrors.Register(SubModuleName, 2, "invalid header height")
)
