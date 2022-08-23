package keeper

import (
	"context"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
)

var _ types.QueryServer = Keeper{}

// CrossChainQuery implements the Query/CrossChainQuery gRPC method
func (k Keeper) CrossChainQuery(context context.Context, query *types.QueryCrossChainQuery) (*types.QueryCrossChainQueryResponse, error) {
	// TODO
	// get queryResult from private store
	return nil, nil
}
