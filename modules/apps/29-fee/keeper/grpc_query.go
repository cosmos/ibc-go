package keeper

import (
	"context"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

var _ types.QueryServer = Keeper{}

// IncentivizedPackets implements the IncentivizedPackets gRPC method
func (k Keeper) IncentivizedPackets(c context.Context, req *types.QueryIncentivizedPacketsRequest) (*types.QueryIncentivizedPacketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c).WithBlockHeight(int64(req.QueryHeight))

	var packets []*types.IdentifiedPacketFee
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.FeeInEscrowPrefix))
	_, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		result := k.MustUnmarshalFee(value)
		packets = append(packets, &result)
		return nil
	})

	if err != nil {
		return nil, status.Error(
			codes.NotFound, err.Error(),
		)
	}

	return &types.QueryIncentivizedPacketsResponse{
		IncentivizedPackets: packets,
	}, nil
}

// IncentivizedPacket implements the IncentivizedPacket gRPC method
func (k Keeper) IncentivizedPacket(c context.Context, req *types.QueryIncentivizedPacketRequest) (*types.QueryIncentivizedPacketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c).WithBlockHeight(int64(req.QueryHeight))

	fee, exists := k.GetFeeInEscrow(ctx, req.PacketId)
	if !exists {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrap(types.ErrFeeNotFound, req.PacketId.String()).Error(),
		)
	}

	return &types.QueryIncentivizedPacketResponse{
		IncentivizedPacket: &fee,
	}, nil
}

// IncentivizedPacketsForChannel implements the IncentivizedPacketsForChannel gRPC method
func (k Keeper) IncentivizedPacketsForChannel(c context.Context, req *types.QueryIncentivizedPacketsForChannelRequest) (*types.QueryIncentivizedPacketsForChannelResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c).WithBlockHeight(int64(req.QueryHeight))

	var packets []*types.IdentifiedPacketFees
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(string(types.KeyFeesInEscrowChannelPrefix(req.PortId, req.ChannelId))+"/"))
	_, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		// the key returned only includes the sequence
		seq, err := strconv.ParseUint(string(key), 10, 64)
		if err != nil {
			return err
		}

		packetID := channeltypes.NewPacketId(req.ChannelId, req.PortId, seq)
		packetFees := k.MustUnmarshalFees(value)

		identifiedPacketFees := types.NewIdentifiedPacketFees(packetID, packetFees.PacketFees)

		packets = append(packets, &identifiedPacketFees)

		return nil
	})

	if err != nil {
		return nil, status.Error(
			codes.NotFound, err.Error(),
		)
	}

	return &types.QueryIncentivizedPacketsForChannelResponse{
		IncentivizedPackets: packets,
	}, nil
}
