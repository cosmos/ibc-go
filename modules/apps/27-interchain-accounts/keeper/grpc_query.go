package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

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

	if strings.TrimSpace(req.CounterpartyPortId) == "" {
		return nil, status.Error(codes.InvalidArgument, "counterparty portID cannot be empty")
	}

	interchainAccountAddress, found := k.GetInterchainAccountAddress(sdk.UnwrapSDKContext(ctx), req.CounterpartyPortId)
	if !found {
		return nil, status.Error(codes.NotFound, sdkerrors.Wrap(types.ErrInterchainAccountNotFound, req.CounterpartyPortId).Error())
	}

	return &types.QueryInterchainAccountAddressResponse{InterchainAccountAddress: interchainAccountAddress}, nil
}
