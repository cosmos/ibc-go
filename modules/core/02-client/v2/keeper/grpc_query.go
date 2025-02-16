package keeper

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
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
func (q queryServer) CounterpartyInfo(goCtx context.Context, req *types.QueryCounterpartyInfoRequest) (*types.QueryCounterpartyInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	info, found := q.GetClientCounterparty(ctx, req.ClientId)
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("client %s counterparty not found", req.ClientId))
	}

	return &types.QueryCounterpartyInfoResponse{CounterpartyInfo: &info}, nil
}
