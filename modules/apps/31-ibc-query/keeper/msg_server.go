package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
)

var _ types.MsgServer = Keeper{}

// SubmitCrossChainQuery Handling SubmitCrossChainQuery transaction
func (k Keeper) SubmitCrossChainQuery(goCtx context.Context, msg *types.MsgSubmitCrossChainQuery) (*types.MsgSubmitCrossChainQueryResponse, error) {
	// TODO
	// 1. UnwrapSDKContext
	// 2. call keeper function
	//   2.1 keeper func transforms msg to query
	//   2.2 keeper func save query in private store
	// 3. emit event or implement emit event function in event.go

	return &types.MsgSubmitCrossChainQueryResponse{}, nil
}

// SubmitCrossChainQueryResult Handling SubmitCrossChainQueryResult transaction
func (k Keeper) SubmitCrossChainQueryResult(goCtx context.Context, msg *types.MsgSubmitCrossChainQueryResult) (*types.MsgSubmitCrossChainQueryResultResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// check CrossChainQuery exist
	if _, found := k.GetCrossChainQuery(ctx, msg.Id); !found {
		return nil, types.ErrCrossChainQueryNotFound
	}

	// remove query from privateStore
	k.DeleteCrossChainQuery(ctx, msg.Id)

	queryResult := &types.CrossChainQueryResult{
		Id:     msg.Id,
		Result: msg.Result,
		Data:   msg.Data,
	}

	// store result in privateStore
	k.SetCrossChainQueryResult(ctx, queryResult)

	return &types.MsgSubmitCrossChainQueryResultResponse{}, nil
}
