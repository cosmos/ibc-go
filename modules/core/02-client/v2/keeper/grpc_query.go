package keeper

import (
	"context"
	"github.com/cosmos/ibc-go/v9/modules/core/02-client/v2/types"
)

var _ types.QueryServer = (*queryServer)(nil)

// queryServer implements the 02-client/v2 types.QueryServer interface.
// It embeds the client keeper to leverage store access while limiting the api of the client keeper.
type queryServer struct {
	*Keeper
}

// NewQueryServer returns a new 02-client/v2 types.QueryServer implementation.
func NewQueryServer(k *Keeper) types.QueryServer {
	return &queryServer{
		Keeper: k,
	}
}
func (q queryServer) CounterpartyInfo(ctx context.Context, request *types.QueryCounterpartyInfoRequest) (*types.QueryCounterpartyInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}
