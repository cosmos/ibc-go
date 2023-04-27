package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
)

// IsHostEnabled retrieves the host enabled boolean from the paramstore.
// True is returned if the host submodule is enabled.
func (k Keeper) IsHostEnabled(ctx sdk.Context) bool {
	var params types.Params
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return false
	}

	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return false
	}

	return params.HostEnabled
}

// GetAllowMessages retrieves the host enabled msg types from the paramstore
func (k Keeper) GetAllowMessages(ctx sdk.Context) []string {
	var params types.Params
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return []string{}
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params.AllowMessages
}

// GetParams returns the total set of the host submodule parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var params types.Params
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.Params{}
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the total set of the host submodule parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)
}
