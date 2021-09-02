package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/router/types"
)

// GetSendEnabled retrieves the send enabled boolean from the paramstore
func (k Keeper) GetFeePercentage(ctx sdk.Context) sdk.Dec {
	var res sdk.Dec
	k.paramSpace.Get(ctx, types.KeyFeePercentage, &res)
	return res
}

// GetParams returns the total set of ibc-transfer parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(k.GetFeePercentage(ctx))
}

// SetParams sets the total set of ibc-transfer parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}
