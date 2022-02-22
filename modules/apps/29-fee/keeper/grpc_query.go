package keeper

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

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

	var identifiedPackets []types.IdentifiedPacketFees
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.FeesInEscrowPrefix))
	_, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		keySplit := strings.Split(string(key), "/")

		channelID, portID := keySplit[2], keySplit[1]
		seq, err := strconv.ParseUint(keySplit[3], 10, 64)
		if err != nil {
			panic(err)
		}

		packetID := channeltypes.NewPacketId(channelID, portID, seq)
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
func (k Keeper) IncentivizedPacket(c context.Context, req *types.QueryIncentivizedPacketRequest) (*types.QueryIncentivizedPacketResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c).WithBlockHeight(int64(req.QueryHeight))

	feesInEscrow, exists := k.GetFeesInEscrow(ctx, req.PacketId)
	if !exists {
		return nil, status.Error(codes.NotFound, sdkerrors.Wrap(types.ErrFeeNotFound, req.PacketId.String()).Error())
	}

	return &types.QueryIncentivizedPacketResponse{
		IncentivizedPacket: types.NewIdentifiedPacketFees(req.PacketId, feesInEscrow.PacketFees),
	}, nil
}
