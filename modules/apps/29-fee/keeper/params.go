package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
)

// GetDistributionAddress retrieves the ics29 fee distribution address from the paramstore.
func (k Keeper) GetDistributionAddress(ctx sdk.Context) string {
	var res string
	k.paramSpace.Get(ctx, types.KeyDistributionAddress, &res)
	return res
}

// GetParams returns the total set of the ics29 fee parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(k.GetDistributionAddress(ctx))
}

// SetParams sets the total set of the ics29 fee parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
