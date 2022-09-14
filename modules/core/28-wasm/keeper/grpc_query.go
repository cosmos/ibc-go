package keeper

import (
	"context"
	"encoding/hex"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
)

var _ types.QueryServer = (*Keeper)(nil)

func (q Keeper) WasmCode(c context.Context, query *types.WasmCodeQuery) (*types.WasmCodeResponse, error) {
	if query == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(q.storeKey)

	codeID, err := hex.DecodeString(query.CodeId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid code id")
	}

	codeKey := types.CodeID(codeID)
	code := store.Get(codeKey)
	if code == nil {
		return nil, status.Error(
			codes.NotFound,
			sdkerrors.Wrap(types.ErrWasmCodeIDNotFound, query.CodeId).Error(),
		)
	}

	return &types.WasmCodeResponse{
		Code: code,
	}, nil
}
