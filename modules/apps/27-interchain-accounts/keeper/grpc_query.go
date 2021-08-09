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
func (k Keeper) InterchainAccountAddress(ctx context.Context, req *types.QueryInterchainAccountAddressRequest) (*types.QueryInterchainAccountAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.CounterpartyPortId == "" {
		return nil, status.Error(codes.InvalidArgument, "counterparty portID cannot be empty")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	interchainAccountAddress, err := k.GetInterchainAccountAddress(sdkCtx, req.CounterpartyPortId)
	if err != nil {
		return nil, err
	}

	return &types.QueryInterchainAccountAddressResponse{InterchainAccountAddress: interchainAccountAddress}, nil
}
