package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/icq/host/types"
)

// IsHostEnabled retrieves the host enabled boolean from the paramstore.
// True is returned if the host submodule is enabled.
func (k Keeper) IsHostEnabled(ctx sdk.Context) bool {
	var res bool
	k.paramSpace.Get(ctx, types.KeyHostEnabled, &res)
	return res
}

// GetAllowHeight retrieves the allow height boolean from the paramstore.
func (k Keeper) GetAllowHeight(ctx sdk.Context) bool {
	var res bool
	k.paramSpace.Get(ctx, types.KeyAllowHeight, &res)
	return res
}

// GetAllowProof retrieves the allow proof boolean from the paramstore.
func (k Keeper) GetAllowProof(ctx sdk.Context) bool {
	var res bool
	k.paramSpace.Get(ctx, types.KeyAllowProof, &res)
	return res
}

// GetAllowQueries retrieves the host enabled query paths from the paramstore
func (k Keeper) GetAllowQueries(ctx sdk.Context) []string {
	var res []string
	k.paramSpace.Get(ctx, types.KeyAllowQueries, &res)
	return res
}

// GetParams returns the total set of the host submodule parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(k.IsHostEnabled(ctx), k.GetAllowHeight(ctx), k.GetAllowProof(ctx), k.GetAllowQueries(ctx))
}

// SetParams sets the total set of the host submodule parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
