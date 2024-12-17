package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// InterchainAccount implements the Query/InterchainAccount gRPC method
func (k Keeper) InterchainAccount(ctx context.Context, req *types.QueryInterchainAccountRequest) (*types.QueryInterchainAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	portID, err := icatypes.NewControllerPortID(req.Owner)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to generate portID from owner address: %s", err)
	}

	addr, found := k.GetInterchainAccountAddress(ctx, req.ConnectionId, portID)
	if !found {
		return nil, status.Errorf(codes.NotFound, "failed to retrieve account address for %s on connection %s", portID, req.ConnectionId)
	}

	return &types.QueryInterchainAccountResponse{
		Address: addr,
	}, nil
}

// Params implements the Query/Params gRPC method
func (k Keeper) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{
		Params: &params,
	}, nil
}
