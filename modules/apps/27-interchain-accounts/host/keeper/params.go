package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/host/types"
)

// GetHostEnabled retrieves the host enabled boolean from the paramstore
func (k Keeper) GetHostEnabled(ctx sdk.Context) bool {
	var res bool
	k.paramSpace.Get(ctx, types.KeyHostEnabled, &res)
	return res
}

// GetAllowMessages retrieves the host enabled msg types from the paramstore
func (k Keeper) GetAllowMessages(ctx sdk.Context) []string {
	var res []string
	k.paramSpace.Get(ctx, types.KeyAllowMessages, &res)
	return res
}

// GetParams returns the total set of the host submodule parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(k.GetHostEnabled(ctx), k.GetAllowMessages(ctx))
}

// SetParams sets the total set of the host submodule parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
