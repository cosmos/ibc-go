package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	errorsmod "cosmossdk.io/errors"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"

	"github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
)

var _ types.QueryServer = (*Keeper)(nil)

// Code implements the Query/Code gRPC method
func (k Keeper) Code(c context.Context, req *types.QueryCodeRequest) (*types.QueryCodeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)

	codeHash, err := hex.DecodeString(req.CodeHash)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid code hash")
	}

	codeKey := types.CodeHashKey(codeHash)
	code := store.Get(codeKey)
	if code == nil {
		return nil, status.Error(codes.NotFound, errorsmod.Wrap(types.ErrWasmCodeHashNotFound, req.CodeHash).Error())
	}

	return &types.QueryCodeResponse{
		Data: code,
	}, nil
}

// CodeHashes implements the Query/CodeHashes gRPC method
func (k Keeper) CodeHashes(c context.Context, req *types.QueryCodeHashesRequest) (*types.QueryCodeHashesResponse, error) {
	var codeHashes []string

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	prefixStore := prefix.NewStore(store, []byte(fmt.Sprintf("%s/", types.KeyCodeHashPrefix)))

	pageRes, err := sdkquery.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, _ []byte, accumulate bool) (bool, error) {
		if accumulate {
			codeHashes = append(codeHashes, string(key))
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryCodeHashesResponse{
		CodeHashes: codeHashes,
		Pagination: pageRes,
	}, nil
}
