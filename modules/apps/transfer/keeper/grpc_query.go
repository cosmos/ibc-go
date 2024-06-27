package keeper

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/v8/internal/validate"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

var (
	_ types.QueryServer   = (*Keeper)(nil)
	_ types.QueryV2Server = (*Keeper)(nil)
)

// Denom implements the Query/Denom gRPC method
func (k Keeper) Denom(c context.Context, req *types.QueryDenomRequest) (*types.QueryDenomResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	hash, err := types.ParseHexHash(strings.TrimPrefix(req.Hash, "ibc/"))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid denom trace hash: %s, error: %s", hash.String(), err))
	}

	ctx := sdk.UnwrapSDKContext(c)
	denom, found := k.GetDenom(ctx, hash)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrap(types.ErrDenomNotFound, req.Hash).Error(),
		)
	}

	return &types.QueryDenomResponse{
		Denom: &denom,
	}, nil
}

// Denoms implements the Query/Denoms gRPC method
func (k Keeper) Denoms(c context.Context, req *types.QueryDenomsRequest) (*types.QueryDenomsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	var denoms types.Denoms
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomKey)

	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		var denom types.Denom
		if err := k.cdc.Unmarshal(value, &denom); err != nil {
			return err
		}

		denoms = append(denoms, denom)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryDenomsResponse{
		Denoms:     denoms.Sort(),
		Pagination: pageRes,
	}, nil
}

// Params implements the Query/Params gRPC method
func (k Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: &params,
	}, nil
}

// DenomHash implements the Query/DenomHash gRPC method
func (k Keeper) DenomHash(c context.Context, req *types.QueryDenomHashRequest) (*types.QueryDenomHashResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// Convert given request trace path to Denom struct to confirm the path in a valid denom trace format
	denom := types.ExtractDenomFromPath(req.Trace)
	if err := denom.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	denomHash := denom.Hash()
	found := k.HasDenom(ctx, denomHash)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrap(types.ErrDenomNotFound, req.Trace).Error(),
		)
	}

	return &types.QueryDenomHashResponse{
		Hash: denomHash.String(),
	}, nil
}

// EscrowAddress implements the EscrowAddress gRPC method
func (k Keeper) EscrowAddress(c context.Context, req *types.QueryEscrowAddressRequest) (*types.QueryEscrowAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	addr := types.GetEscrowAddress(req.PortId, req.ChannelId)

	if err := validate.GRPCRequest(req.PortId, req.ChannelId); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(c)
	if !k.channelKeeper.HasChannel(ctx, req.PortId, req.ChannelId) {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", req.PortId, req.ChannelId).Error(),
		)
	}

	return &types.QueryEscrowAddressResponse{
		EscrowAddress: addr.String(),
	}, nil
}

// TotalEscrowForDenom implements the TotalEscrowForDenom gRPC method.
func (k Keeper) TotalEscrowForDenom(c context.Context, req *types.QueryTotalEscrowForDenomRequest) (*types.QueryTotalEscrowForDenomResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	if err := sdk.ValidateDenom(req.Denom); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	amount := k.GetTotalEscrowForDenom(ctx, req.Denom)

	return &types.QueryTotalEscrowForDenomResponse{
		Amount: amount,
	}, nil
}
