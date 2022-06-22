package keeper

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v3/modules/apps/nft-transfer/types"
)

var _ types.QueryServer = Keeper{}

// ClassTrace implements the Query/ClassTrace gRPC method
func (k Keeper) ClassTrace(c context.Context,
	req *types.QueryClassTraceRequest,
) (*types.QueryClassTraceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	hash, err := types.ParseHexHash(strings.TrimPrefix(req.Hash, "ibc/"))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid denom trace hash: %s, error: %s", hash.String(), err))
	}

	ctx := sdk.UnwrapSDKContext(c)
	classTrace, found := k.GetClassTrace(ctx, hash)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrap(types.ErrTraceNotFound, req.Hash).Error(),
		)
	}

	return &types.QueryClassTraceResponse{
		ClassTrace: &classTrace,
	}, nil
}

// ClassTraces implements the Query/ClassTraces gRPC method
func (k Keeper) ClassTraces(c context.Context,
	req *types.QueryClassTracesRequest,
) (*types.QueryClassTracesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	traces := types.Traces{}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ClassTraceKey)
	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		result, err := k.UnmarshalClassTrace(value)
		if err != nil {
			return err
		}

		traces = append(traces, result)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryClassTracesResponse{
		ClassTraces: traces.Sort(),
		Pagination:  pageRes,
	}, nil
}

// ClassHash implements the Query/ClassHash gRPC method
func (k Keeper) ClassHash(c context.Context,
	req *types.QueryClassHashRequest,
) (*types.QueryClassHashResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// Convert given request trace path to ClassTrace struct to confirm the path in a valid class trace format
	classTrace := types.ParseClassTrace(req.Trace)
	if err := classTrace.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	classHash := classTrace.Hash()
	found := k.HasClassTrace(ctx, classHash)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrap(types.ErrTraceNotFound, req.Trace).Error(),
		)
	}

	return &types.QueryClassHashResponse{
		Hash: classHash.String(),
	}, nil
}

// EscrowAddress implements the EscrowAddress gRPC method
func (k Keeper) EscrowAddress(c context.Context,
	req *types.QueryEscrowAddressRequest,
) (*types.QueryEscrowAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	addr := types.GetEscrowAddress(req.PortId, req.ChannelId)

	return &types.QueryEscrowAddressResponse{
		EscrowAddress: addr.String(),
	}, nil
}
