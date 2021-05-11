package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	"github.com/cosmos/ibc-go/modules/core/28-wasm/types"
)

var _ types.QueryServer = (*Keeper)(nil)

func (q Keeper) LatestWASMCode(c context.Context, query *types.LatestWASMCodeQuery) (*types.LatestWASMCodeResponse, error) {
	if query == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if query.ClientType == "" {
		return nil, status.Error(codes.InvalidArgument, "empty client type string")
	}

	clientType := query.ClientType

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(q.storeKey)

	latestCodeKey := host.LatestWASMCode(clientType)
	latestCodeId := store.Get(latestCodeKey)
	if latestCodeId == nil {
		return nil, status.Error(codes.NotFound, "no code has been uploaded till now.")
	}

	return &types.LatestWASMCodeResponse{
		Code: store.Get(host.WASMCode(clientType, string(latestCodeId))),
	}, nil
}

func (q Keeper) LatestWASMCodeEntry(c context.Context, query *types.LatestWASMCodeEntryQuery) (*types.LatestWASMCodeEntryResponse, error) {
	if query == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if query.ClientType == "" {
		return nil, status.Error(codes.InvalidArgument, "empty client type string")
	}

	clientType := query.ClientType

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(q.storeKey)
	latestCodeKey := host.LatestWASMCode(clientType)
	latestCodeId := store.Get(latestCodeKey)
	if latestCodeId == nil {
		return nil, status.Error(codes.NotFound, "no code has been uploaded till now.")
	}

	bz := store.Get(host.WASMCodeEntry(clientType, string(latestCodeId)))
	var entry types.WasmCodeEntry
	if err := q.cdc.Unmarshal(bz, &entry); err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.LatestWASMCodeEntryResponse{
		CodeId: string(latestCodeId),
		Entry:  &entry,
	}, nil
}
