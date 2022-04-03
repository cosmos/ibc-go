package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
)

var _ types.QueryServer = Keeper{}

// IncentivizedPackets implements the IncentivizedPackets gRPC method
func (k Keeper) IncentivizedPackets(goCtx context.Context, req *types.QueryIncentivizedPacketsRequest) (*types.QueryIncentivizedPacketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx).WithBlockHeight(int64(req.QueryHeight))

	var identifiedPackets []types.IdentifiedPacketFees
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.FeesInEscrowPrefix))
	_, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		packetID, err := types.ParseKeyFeesInEscrow(types.FeesInEscrowPrefix + string(key))
		if err != nil {
			return err
		}

		packetFees := k.MustUnmarshalFees(value)
		identifiedPackets = append(identifiedPackets, types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees))
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryIncentivizedPacketsResponse{
		IncentivizedPackets: identifiedPackets,
	}, nil
}

// IncentivizedPacket implements the IncentivizedPacket gRPC method
func (k Keeper) IncentivizedPacket(goCtx context.Context, req *types.QueryIncentivizedPacketRequest) (*types.QueryIncentivizedPacketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx).WithBlockHeight(int64(req.QueryHeight))

	feesInEscrow, exists := k.GetFeesInEscrow(ctx, req.PacketId)
	if !exists {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrapf(types.ErrFeeNotFound, "channel: %s, port: %s, sequence: %d", req.PacketId.ChannelId, req.PacketId.PortId, req.PacketId.Sequence).Error())
	}

	return &types.QueryIncentivizedPacketResponse{
		IncentivizedPacket: types.NewIdentifiedPacketFees(req.PacketId, feesInEscrow.PacketFees),
	}, nil
}

// IncentivizedPacketsForChannel implements the IncentivizedPacketsForChannel gRPC method
func (k Keeper) IncentivizedPacketsForChannel(goCtx context.Context, req *types.QueryIncentivizedPacketsForChannelRequest) (*types.QueryIncentivizedPacketsForChannelResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx).WithBlockHeight(int64(req.QueryHeight))

	var packets []*types.IdentifiedPacketFees
	keyPrefix := types.KeyFeesInEscrowChannelPrefix(req.PortId, req.ChannelId)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), keyPrefix)
	_, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		packetID, err := types.ParseKeyFeesInEscrow(string(keyPrefix) + string(key))
		if err != nil {
			return err
		}

		packetFees := k.MustUnmarshalFees(value)

		identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)
		packets = append(packets, &identifiedPacketFees)

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryIncentivizedPacketsForChannelResponse{
		IncentivizedPackets: packets,
	}, nil
}

// TotalRecvFees implements the Query/TotalRecvFees gRPC method
func (k Keeper) TotalRecvFees(goCtx context.Context, req *types.QueryTotalRecvFeesRequest) (*types.QueryTotalRecvFeesResponse, error) {

	ctx := sdk.UnwrapSDKContext(goCtx)

	feesInEscrow, found := k.GetFeesInEscrow(ctx, req.PacketId)
	if !found {
		return nil, status.Errorf(
			codes.NotFound,
			sdkerrors.Wrapf(types.ErrFeeNotFound, "channel: %s, port: %s, sequence: %d", req.PacketId.ChannelId, req.PacketId.PortId, req.PacketId.Sequence).Error(),
		)
	}

	var recvFees sdk.Coins
	for _, packetFee := range feesInEscrow.PacketFees {
		recvFees = recvFees.Add(packetFee.Fee.RecvFee...)
	}

	return &types.QueryTotalRecvFeesResponse{
		RecvFees: recvFees,
	}, nil
}

// TotalAckFees implements the Query/TotalAckFees gRPC method
func (k Keeper) TotalAckFees(goCtx context.Context, req *types.QueryTotalAckFeesRequest) (*types.QueryTotalAckFeesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	feesInEscrow, found := k.GetFeesInEscrow(ctx, req.PacketId)
	if !found {
		return nil, status.Errorf(
			codes.NotFound,
			sdkerrors.Wrapf(types.ErrFeeNotFound, "channel: %s, port: %s, sequence: %d", req.PacketId.ChannelId, req.PacketId.PortId, req.PacketId.Sequence).Error(),
		)
	}

	var ackFees sdk.Coins
	for _, packetFee := range feesInEscrow.PacketFees {
		ackFees = ackFees.Add(packetFee.Fee.AckFee...)
	}

	return &types.QueryTotalAckFeesResponse{
		AckFees: ackFees,
	}, nil
}

// TotalTimeoutFees implements the Query/TotalTimeoutFees gRPC method
func (k Keeper) TotalTimeoutFees(goCtx context.Context, req *types.QueryTotalTimeoutFeesRequest) (*types.QueryTotalTimeoutFeesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	feesInEscrow, found := k.GetFeesInEscrow(ctx, req.PacketId)
	if !found {
		return nil, status.Errorf(
			codes.NotFound,
			sdkerrors.Wrapf(types.ErrFeeNotFound, "channel: %s, port: %s, sequence: %d", req.PacketId.ChannelId, req.PacketId.PortId, req.PacketId.Sequence).Error(),
		)
	}

	var timeoutFees sdk.Coins
	for _, packetFee := range feesInEscrow.PacketFees {
		timeoutFees = timeoutFees.Add(packetFee.Fee.TimeoutFee...)
	}

	return &types.QueryTotalTimeoutFeesResponse{
		TimeoutFees: timeoutFees,
	}, nil
}
