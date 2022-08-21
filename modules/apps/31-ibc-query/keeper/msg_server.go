package keeper

import (
	"context"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
)

var _ types.MsgServer = Keeper{}

// SubmitCrossChainQuery Handling CrossChainQuery transaction
func (k Keeper) SubmitCrossChainQuery(goCtx context.Context, msg *types.MsgSubmitCrossChainQuery) (*types.MsgSubmitCrossChainQueryResponse, error) {
	// TODO
	// 1. UnwrapSDKContext
	// 2. call keeper function
	//   2.1 keeper func transforms msg to query
	//   2.2 keeper func save query in private store
	// 3. emit event or implement emit event function in event.go

	return &types.MsgSubmitCrossChainQueryResponse{}, nil
}

func (k Keeper) SubmitCrossChainQueryResult(goCtx context.Context, msg *types.MsgSubmitCrossChainQueryResult) (*types.MsgSubmitCrossChainQueryResultResponse, error) {
	// TODO
	// 0. verify the result using local client <- other function
	// 1. retrieve the query from privateStore
	// 2. remove query from privateStore
	// 3. store result in privateStore

	ctx := sdk.UnwrapSDKContext(goCtx)

	_ = prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryKey)
	_ = prefix.NewStore(ctx.KVStore(k.storeKey), types.QueryResultKey)

	// retrieve the query from privateStore
	// k.RetrieveQuery()
	// k.StoreQueryResult()

	return &types.MsgSubmitCrossChainQueryResultResponse{}, nil
}
