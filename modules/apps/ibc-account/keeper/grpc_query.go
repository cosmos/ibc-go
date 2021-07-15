package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/ibc-go/modules/apps/ibc-account/types"
)

var _ types.QueryServer = Keeper{}

// IBCAccount implements the Query/IBCAccount gRPC method
func (k Keeper) IBCAccount(ctx context.Context, req *types.QueryIBCAccountRequest) (*types.QueryIBCAccountResponse, error) {
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

	return &types.QueryIBCAccountResponse{AccountAddress: address}, nil
}
