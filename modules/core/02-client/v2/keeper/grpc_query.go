package keeper

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// CounterpartyInfo gets the CounterpartyInfo from the store corresponding to the request client ID.
func (q queryServer) CounterpartyInfo(ctx context.Context, request *types.QueryCounterpartyInfoRequest) (*types.QueryCounterpartyInfoResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	info, found := q.GetClientCounterparty(sdkCtx, request.ClientId)
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("client %s counterparty not found", request.ClientId))
	}

	return &types.QueryCounterpartyInfoResponse{CounterpartyInfo: &info}, nil
}
