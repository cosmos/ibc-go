package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

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

	if err := host.ChannelIdentifierValidator(req.ChannelId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	creator, foundCreator := q.GetCreator(ctx, req.ChannelId)
	channel, foundChannel := q.GetChannel(ctx, req.ChannelId)

	if !foundCreator && !foundChannel {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrChannelNotFound, "channel-id: %s", req.ChannelId).Error(),
		)
	}

	return types.NewQueryChannelResponse(creator, channel), nil
}

// PacketCommitment implements the Query/PacketCommitment gRPC method.
func (q *queryServer) PacketCommitment(ctx context.Context, req *types.QueryPacketCommitmentRequest) (*types.QueryPacketCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ChannelIdentifierValidator(req.ChannelId); err != nil {
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

	return types.NewQueryPacketCommitmentResponse(commitment, nil, clienttypes.GetSelfHeight(ctx)), nil
}

// PacketAcknowledgement implements the Query/PacketAcknowledgement gRPC method.
func (q *queryServer) PacketAcknowledgement(ctx context.Context, req *types.QueryPacketAcknowledgementRequest) (*types.QueryPacketAcknowledgementResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ChannelIdentifierValidator(req.ChannelId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.Sequence == 0 {
		return nil, status.Error(codes.InvalidArgument, "packet sequence cannot be 0")
	}

	if !q.HasChannel(ctx, req.ChannelId) {
		return nil, status.Error(codes.NotFound, errorsmod.Wrap(types.ErrChannelNotFound, req.ChannelId).Error())
	}

	acknowledgement := q.GetPacketAcknowledgement(ctx, req.ChannelId, req.Sequence)
	if len(acknowledgement) == 0 {
		return nil, status.Error(codes.NotFound, "packet acknowledgement hash not found")
	}

	return types.NewQueryPacketAcknowledgementResponse(acknowledgement, nil, clienttypes.GetSelfHeight(ctx)), nil
}

// PacketReceipt implements the Query/PacketReceipt gRPC method.
func (q *queryServer) PacketReceipt(ctx context.Context, req *types.QueryPacketReceiptRequest) (*types.QueryPacketReceiptResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ChannelIdentifierValidator(req.ChannelId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.Sequence == 0 {
		return nil, status.Error(codes.InvalidArgument, "packet sequence cannot be 0")
	}

	if !q.HasChannel(ctx, req.ChannelId) {
		return nil, status.Error(codes.NotFound, errorsmod.Wrap(types.ErrChannelNotFound, req.ChannelId).Error())
	}

	hasReceipt := q.HasPacketReceipt(ctx, req.ChannelId, req.Sequence)

	return types.NewQueryPacketReceiptResponse(hasReceipt, nil, clienttypes.GetSelfHeight(ctx)), nil
}
