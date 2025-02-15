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

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
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

// NextSequenceSend implements the Query/NextSequenceSend gRPC method
func (q *queryServer) NextSequenceSend(goCtx context.Context, req *types.QueryNextSequenceSendRequest) (*types.QueryNextSequenceSendResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	sequence, found := q.GetNextSequenceSend(ctx, req.ClientId)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(types.ErrSequenceSendNotFound, "client-id %s", req.ClientId).Error(),
		)
	}
	return types.NewQueryNextSequenceSendResponse(sequence, nil, clienttypes.GetSelfHeight(ctx)), nil
}

// PacketCommitment implements the Query/PacketCommitment gRPC method.
func (q *queryServer) PacketCommitment(goCtx context.Context, req *types.QueryPacketCommitmentRequest) (*types.QueryPacketCommitmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.Sequence == 0 {
		return nil, status.Error(codes.InvalidArgument, "packet sequence cannot be 0")
	}

	commitment := q.GetPacketCommitment(ctx, req.ClientId, req.Sequence)
	if len(commitment) == 0 {
		return nil, status.Error(codes.NotFound, "packet commitment hash not found")
	}

	return types.NewQueryPacketCommitmentResponse(commitment, nil, clienttypes.GetSelfHeight(ctx)), nil
}

// PacketCommitments implements the Query/PacketCommitments gRPC method
func (q *queryServer) PacketCommitments(goCtx context.Context, req *types.QueryPacketCommitmentsRequest) (*types.QueryPacketCommitmentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var commitments []*types.PacketState
	store := prefix.NewStore(runtime.KVStoreAdapter(q.storeService.OpenKVStore(goCtx)), hostv2.PacketCommitmentPrefixKey(req.ClientId))

	pageRes, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		keySplit := strings.Split(string(key), "/")

		sequence := sdk.BigEndianToUint64([]byte(keySplit[len(keySplit)-1]))
		if sequence == 0 {
			return types.ErrInvalidPacket
		}

		commitment := types.NewPacketState(req.ClientId, sequence, value)
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
func (q *queryServer) PacketAcknowledgement(goCtx context.Context, req *types.QueryPacketAcknowledgementRequest) (*types.QueryPacketAcknowledgementResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.Sequence == 0 {
		return nil, status.Error(codes.InvalidArgument, "packet sequence cannot be 0")
	}

	acknowledgement := q.GetPacketAcknowledgement(ctx, req.ClientId, req.Sequence)
	if len(acknowledgement) == 0 {
		return nil, status.Error(codes.NotFound, "packet acknowledgement hash not found")
	}

	return types.NewQueryPacketAcknowledgementResponse(acknowledgement, nil, clienttypes.GetSelfHeight(ctx)), nil
}

// PacketAcknowledgements implements the Query/PacketAcknowledgements gRPC method.
func (q *queryServer) PacketAcknowledgements(goCtx context.Context, req *types.QueryPacketAcknowledgementsRequest) (*types.QueryPacketAcknowledgementsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var acks []*types.PacketState
	store := prefix.NewStore(runtime.KVStoreAdapter(q.storeService.OpenKVStore(goCtx)), hostv2.PacketAcknowledgementPrefixKey(req.ClientId))

	// if a list of packet sequences is provided then query for each specific ack and return a list <= len(req.PacketCommitmentSequences)
	// otherwise, maintain previous behaviour and perform paginated query
	for _, seq := range req.PacketCommitmentSequences {
		acknowledgement := q.GetPacketAcknowledgement(ctx, req.ClientId, seq)
		if len(acknowledgement) == 0 {
			continue
		}

		ack := types.NewPacketState(req.ClientId, seq, acknowledgement)
		acks = append(acks, &ack)
	}

	if len(req.PacketCommitmentSequences) > 0 {
		selfHeight := clienttypes.GetSelfHeight(ctx)
		return &types.QueryPacketAcknowledgementsResponse{
			Acknowledgements: acks,
			Pagination:       nil,
			Height:           selfHeight,
		}, nil
	}

	pageRes, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		keySplit := strings.Split(string(key), "/")

		sequence := sdk.BigEndianToUint64([]byte(keySplit[len(keySplit)-1]))
		if sequence == 0 {
			return types.ErrInvalidPacket
		}

		ack := types.NewPacketState(req.ClientId, sequence, value)
		acks = append(acks, &ack)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryPacketAcknowledgementsResponse{
		Acknowledgements: acks,
		Pagination:       pageRes,
		Height:           clienttypes.GetSelfHeight(ctx),
	}, nil
}

// PacketReceipt implements the Query/PacketReceipt gRPC method.
func (q *queryServer) PacketReceipt(goCtx context.Context, req *types.QueryPacketReceiptRequest) (*types.QueryPacketReceiptResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if req.Sequence == 0 {
		return nil, status.Error(codes.InvalidArgument, "packet sequence cannot be 0")
	}

	hasReceipt := q.HasPacketReceipt(ctx, req.ClientId, req.Sequence)

	return types.NewQueryPacketReceiptResponse(hasReceipt, nil, clienttypes.GetSelfHeight(ctx)), nil
}

// UnreceivedPackets implements the Query/UnreceivedPackets gRPC method. Given
// a list of counterparty packet commitments, the querier checks if the packet
// has already been received by checking if a receipt exists on this
// chain for the packet sequence. All packets that haven't been received yet
// are returned in the response
// Usage: To use this method correctly, first query all packet commitments on
// the sending chain using the Query/PacketCommitments gRPC method.
// Then input the returned sequences into the QueryUnreceivedPacketsRequest
// and send the request to this Query/UnreceivedPackets on the **receiving**
// chain. This gRPC method will then return the list of packet sequences that
// are yet to be received on the receiving chain.
//
// NOTE: The querier makes the assumption that the provided list of packet
// commitments is correct and will not function properly if the list
// is not up to date. Ideally the query height should equal the latest height
// on the counterparty's client which represents this chain.
func (q *queryServer) UnreceivedPackets(goCtx context.Context, req *types.QueryUnreceivedPacketsRequest) (*types.QueryUnreceivedPacketsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var unreceivedSequences []uint64
	for i, seq := range req.Sequences {
		// filter for invalid sequences to ensure they are not included in the response value.
		if seq == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "packet sequence %d cannot be 0", i)
		}

		// if the packet receipt does not exist, then it is unreceived
		if !q.HasPacketReceipt(ctx, req.ClientId, seq) {
			unreceivedSequences = append(unreceivedSequences, seq)
		}
	}

	selfHeight := clienttypes.GetSelfHeight(ctx)
	return &types.QueryUnreceivedPacketsResponse{
		Sequences: unreceivedSequences,
		Height:    selfHeight,
	}, nil
}

// UnreceivedAcks implements the Query/UnreceivedAcks gRPC method. Given
// a list of counterparty packet acknowledgements, the querier checks if the packet
// has already been received by checking if the packet commitment still exists on this
// chain (original sender) for the packet sequence.
// All acknowledgmeents that haven't been received yet are returned in the response.
// Usage: To use this method correctly, first query all packet acknowledgements on
// the original receiving chain (ie the chain that wrote the acks) using the Query/PacketAcknowledgements gRPC method.
// Then input the returned sequences into the QueryUnreceivedAcksRequest
// and send the request to this Query/UnreceivedAcks on the **original sending**
// chain. This gRPC method will then return the list of packet sequences whose
// acknowledgements are already written on the receiving chain but haven't yet
// been received back to the sending chain.
//
// NOTE: The querier makes the assumption that the provided list of packet
// acknowledgements is correct and will not function properly if the list
// is not up to date. Ideally the query height should equal the latest height
// on the counterparty's client which represents this chain.
func (q *queryServer) UnreceivedAcks(goCtx context.Context, req *types.QueryUnreceivedAcksRequest) (*types.QueryUnreceivedAcksResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := host.ClientIdentifierValidator(req.ClientId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var unreceivedSequences []uint64

	for _, seq := range req.PacketAckSequences {
		if seq == 0 {
			return nil, status.Error(codes.InvalidArgument, "packet sequence cannot be 0")
		}

		// if packet commitment still exists on the original sending chain, then packet ack has not been received
		// since processing the ack will delete the packet commitment
		if commitment := q.GetPacketCommitment(ctx, req.ClientId, seq); len(commitment) != 0 {
			unreceivedSequences = append(unreceivedSequences, seq)
		}

	}

	selfHeight := clienttypes.GetSelfHeight(ctx)
	return &types.QueryUnreceivedAcksResponse{
		Sequences: unreceivedSequences,
		Height:    selfHeight,
	}, nil
}
