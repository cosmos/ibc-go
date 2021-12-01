package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/controller/types"
)

// GetControllerEnabled retrieves the host enabled boolean from the paramstore
func (k Keeper) GetControllerEnabled(ctx sdk.Context) bool {
	var res bool
	k.paramSpace.Get(ctx, types.KeyControllerEnabled, &res)
	return res
}

// GetParams returns the total set of the host submodule parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(k.GetControllerEnabled(ctx))
}

// SetParams sets the total set of the host submodule parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
