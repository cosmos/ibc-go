package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

var _ types.QueryServer = (*queryServer)(nil)

// queryServer implements the packet-server types.QueryServer interface.
type queryServer struct {
	*Keeper
}

// NewQueryServer returns a new types.QueryServer implementation.
func NewQueryServer(k *Keeper) types.QueryServer {
	return &queryServer{
		Keeper: k,
	}
}

// Client implements the Query/Client gRPC method
func (q *queryServer) Client(ctx context.Context, req *types.QueryClientRequest) (*types.QueryClientResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	res := types.QueryClientResponse{}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator, foundCreator := q.GetCreator(sdkCtx, req.ClientId)
	counterparty, foundCounterparty := q.GetCounterparty(sdkCtx, req.ClientId)

	if !foundCreator && !foundCounterparty {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrCounterpartyNotFound, "client-id: %s", req.ClientId).Error(),
		)
	}

	res.Counterparty = counterparty
	res.Creator = creator

	return &res, nil
}
