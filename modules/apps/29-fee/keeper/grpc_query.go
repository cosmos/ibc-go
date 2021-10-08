package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
)

var _ types.QueryServer = Keeper{}

// ReceiveFee implements the ReceiveFee gRPC method
func (q Keeper) ReceiveFee(c context.Context, req *types.QueryReceiveFeeRequest) (*types.QueryReceiveFeeResponse, error) {

	return &types.QueryReceiveFeeResponse{}, nil
}

// AckFee implements the AckFee gRPC method
func (q Keeper) AckFee(c context.Context, req *types.QueryAckFeeRequest) (*types.QueryAckFeeResponse, error) {

	return &types.QueryAckFeeResponse{}, nil
}

// TimeoutFee implements the TimeoutFee gRPC method
func (q Keeper) TimeoutFee(c context.Context, req *types.QueryTimeoutFeeRequest) (*types.QueryTimeoutFeeResponse, error) {

	return &types.QueryTimeoutFeeResponse{}, nil
}

// IncentivizedPackets implements the IncentivizedPackets gRPC method
func (q Keeper) IncentivizedPackets(c context.Context, req *types.QueryIncentivizedPacketsRequest) (*types.QueryIncentivizedPacketsResponse, error) {

	return &types.QueryIncentivizedPacketsResponse{}, nil
}

// IncentivizedPacket implements the IncentivizedPacket gRPC method
func (q Keeper) IncentivizedPacket(c context.Context, req *types.QueryIncentivizedPacketRequest) (*types.QueryIncentivizedPacketResponse, error) {

	return &types.QueryIncentivizedPacketResponse{}, nil
}
