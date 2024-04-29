package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
)

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	GetParamSet(ctx sdk.Context, ps paramtypes.ParamSet)
}

// ScopedIBCKeeper is a type alias of x/capability's ScopedKeeper used for depinject module outputs.
type ScopedIBCKeeper capabilitykeeper.ScopedKeeper
