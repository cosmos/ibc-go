package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// Params implements the Query/Params gRPC method
func (k Keeper) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{
		Params: &params,
	}, nil
}
