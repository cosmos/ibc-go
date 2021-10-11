package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

// ReceiveFee implements the ReceiveFee gRPC method
func (q Keeper) ReceiveFee(c context.Context, req *types.QueryReceiveFeeRequest) (*types.QueryReceiveFeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)
	q.GetFeeInEscrow(ctx, req.)
	return &types.QueryReceiveFeeResponse{}, nil
}

// AckFee implements the AckFee gRPC method
func (q Keeper) AckFee(c context.Context, req *types.QueryAckFeeRequest) (*types.QueryAckFeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryAckFeeResponse{}, nil
}

// TimeoutFee implements the TimeoutFee gRPC method
func (q Keeper) TimeoutFee(c context.Context, req *types.QueryTimeoutFeeRequest) (*types.QueryTimeoutFeeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryTimeoutFeeResponse{}, nil
}

// IncentivizedPackets implements the IncentivizedPackets gRPC method
func (q Keeper) IncentivizedPackets(c context.Context, req *types.QueryIncentivizedPacketsRequest) (*types.QueryIncentivizedPacketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryIncentivizedPacketsResponse{}, nil
}

// IncentivizedPacket implements the IncentivizedPacket gRPC method
func (q Keeper) IncentivizedPacket(c context.Context, req *types.QueryIncentivizedPacketRequest) (*types.QueryIncentivizedPacketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryIncentivizedPacketResponse{}, nil
}
