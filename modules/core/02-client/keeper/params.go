package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// GetAllowedClients retrieves the allowed clients from the params stored.
func (k Keeper) GetAllowedClients(ctx sdk.Context) []string {
	p := k.GetParams(ctx)
	return p.AllowedClients
}

// GetParams returns the total set of ibc-client parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	var p types.Params
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte(types.ParamsKey))
	if bz == nil {
		return p
	}

	k.cdc.MustUnmarshal(bz, &p)
	return p
}

// SetParams sets the total set of ibc-client parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set([]byte(types.ParamsKey), bz)
}
