package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

var _ types.QueryServer = (*queryServer)(nil)

// queryServer implements the channel/v2 types.QueryServer interface.
type queryServer struct {
	*Keeper
}

// NewQueryServer returns a new types.QueryServer implementation.
func NewQueryServer(k *Keeper) types.QueryServer {
	return &queryServer{
		Keeper: k,
	}
}

// Channel implements the Query/Channel gRPC method
func (q *queryServer) Channel(ctx context.Context, req *types.QueryChannelRequest) (*types.QueryChannelResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ClientIdentifierValidator(req.ChannelId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	res := types.QueryChannelResponse{}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	creator, foundCreator := q.GetCreator(sdkCtx, req.ChannelId)
	channel, foundChannel := q.GetChannel(sdkCtx, req.ChannelId)

	if !foundCreator && !foundChannel {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrChannelNotFound, "channel-id: %s", req.ChannelId).Error(),
		)
	}

	res.Channel = channel
	res.Creator = creator

	return &res, nil
}

// PacketCommitment implements the Query/PacketCommitment gRPC method.
func (q *queryServer) PacketCommitment(ctx context.Context, req *types.QueryPacketCommitmentRequest) (*types.QueryPacketCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ClientIdentifierValidator(req.ChannelId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.Sequence == 0 {
		return nil, status.Error(codes.InvalidArgument, "packet sequence cannot be 0")
	}

	if !q.HasChannel(ctx, req.ChannelId) {
		return nil, status.Error(codes.NotFound, errorsmod.Wrap(types.ErrChannelNotFound, req.ChannelId).Error())
	}

	commitment := q.GetPacketCommitment(ctx, req.ChannelId, req.Sequence)
	if len(commitment) == 0 {
		return nil, status.Error(codes.NotFound, "packet commitment hash not found")
	}

	return &types.QueryPacketCommitmentResponse{
		Commitment:  commitment,
		Proof:       nil,
		ProofHeight: clienttypes.GetSelfHeight(ctx),
	}, nil
}
