package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/prototypes/x/tokenfactory/types"
)

// GetParams gets the tokenfactory module's parameters.
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	params, err := k.ParamsStore.Get(ctx)
	if err != nil {
		return types.Params{}, err
	}
	return params, nil
}

// SetParams sets the tokenfactory module's parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.ParamsStore.Set(ctx, params)
}
