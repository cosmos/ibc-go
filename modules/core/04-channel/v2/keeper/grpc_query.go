package keeper

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	hostv2 "github.com/cosmos/ibc-go/v9/modules/core/24-host/v2"
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

	channel, found := q.GetChannel(ctx, req.ChannelId)
	if !found {
		return nil, status.Error(codes.NotFound, errorsmod.Wrapf(types.ErrChannelNotFound, "channel-id: %s", req.ChannelId).Error())
	}

	return types.NewQueryChannelResponse(channel), nil
}

// NextSequenceSend implements the Query/NextSequenceSend gRPC method
func (q *queryServer) NextSequenceSend(ctx context.Context, req *types.QueryNextSequenceSendRequest) (*types.QueryNextSequenceSendResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ChannelIdentifierValidator(req.ChannelId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	sequence, found := q.GetNextSequenceSend(ctx, req.ChannelId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrSequenceSendNotFound, "channel-id %s", req.ChannelId).Error(),
		)
	}
	return types.NewQueryNextSequenceSendResponse(sequence, nil, clienttypes.GetSelfHeight(ctx)), nil
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

// PacketCommitments implements the Query/PacketCommitments gRPC method
func (q *queryServer) PacketCommitments(ctx context.Context, req *types.QueryPacketCommitmentsRequest) (*types.QueryPacketCommitmentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := host.ChannelIdentifierValidator(req.ChannelId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if !q.HasChannel(ctx, req.ChannelId) {
		return nil, status.Error(codes.NotFound, errorsmod.Wrap(types.ErrChannelNotFound, req.ChannelId).Error())
	}

	var commitments []*types.PacketState
	store := prefix.NewStore(runtime.KVStoreAdapter(q.storeService.OpenKVStore(ctx)), hostv2.PacketCommitmentPrefixKey(req.ChannelId))

	pageRes, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		keySplit := strings.Split(string(key), "/")

		sequence := sdk.BigEndianToUint64([]byte(keySplit[len(keySplit)-1]))
		if sequence == 0 {
			return types.ErrInvalidPacket
		}

		commitment := types.NewPacketState(req.ChannelId, sequence, value)
		commitments = append(commitments, &commitment)
		return nil
	})
	if err != nil {
		return nil, err
	}

	selfHeight := clienttypes.GetSelfHeight(ctx)
	return &types.QueryPacketCommitmentsResponse{
		Commitments: commitments,
		Pagination:  pageRes,
		Height:      selfHeight,
	}, nil
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
