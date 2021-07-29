package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
)

var _ types.QueryServer = Keeper{}

// InterchainAccount implements the Query/InterchainAccount gRPC method
func (k Keeper) InterchainAccount(ctx context.Context, req *types.QueryInterchainAccountRequest) (*types.QueryInterchainAccountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address cannot be empty")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	portId := k.GeneratePortId(req.Address, req.ConnectionId)

	address, err := k.GetInterchainAccountAddress(sdkCtx, portId)
	if err != nil {
		return nil, err
	}

	return &types.QueryInterchainAccountResponse{AccountAddress: address}, nil
}
