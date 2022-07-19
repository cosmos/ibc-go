package keeper

import (
	"context"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
)

var _ types.MsgServer = Keeper{}

func (k Keeper) SubmitCrossChainQuery(goCtx context.Context, msg *types.MsgCrossChainQuery) (*types.MsgCrossChainQueryResponse, error) {
	return nil, nil
}

func (k Keeper) SubmitCrossChainQueryResult(goCtx context.Context, msg *types.MsgCrossChainQueryResult) (*types.MsgCrossChainQueryResultResponse, error) {
	return nil, nil
}
