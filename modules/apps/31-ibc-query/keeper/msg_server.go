package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v4/modules/apps/31-ibc-query/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
)

var _ types.MsgServer = Keeper{}

// SubmitCrossChainQuery Handling SubmitCrossChainQuery transaction
func (k Keeper) SubmitCrossChainQuery(goCtx context.Context, msg *types.MsgSubmitCrossChainQuery) (*types.MsgSubmitCrossChainQueryResponse, error) {
	// UnwrapSDKContext
	ctx := sdk.UnwrapSDKContext(goCtx)

	currentTimestamp := uint64(ctx.BlockTime().UnixNano())
	currentHeight := clienttypes.GetSelfHeight(ctx)

	// Sanity-check that localTimeoutHeight is 0 or greater than the current height, otherwise the query will always time out.
	if !(msg.LocalTimeoutHeight == 0 || msg.LocalTimeoutHeight > currentHeight.RevisionHeight){
		return nil, sdkerrors.Wrapf(
			types.ErrInvalidTimeoutHeight,
			"localTimeoutHeight is not 0 and current height >= localTimeoutHeight(%d >= %d)", currentHeight.RevisionHeight, msg.LocalTimeoutHeight,
		)
	}
	// Sanity-check that localTimeoutTimestamp is 0 or greater than the current timestamp, otherwise the query will always time out.
	if !(msg.LocalTimeoutStamp == 0 || msg.LocalTimeoutStamp > currentTimestamp){
		return nil, sdkerrors.Wrapf(
			types.ErrQuerytTimeout,
			"localTimeoutTimestamp is not 0 and current timestamp >= localTimeoutTimestamp(%d >= %d)", currentTimestamp, msg.LocalTimeoutStamp,
		)
	}


	// call keeper function
	// keeper func save query in private store
	query := types.MsgSubmitCrossChainQuery{
		Id: msg.Id,
		Path: msg.Path,
		LocalTimeoutHeight: msg.LocalTimeoutHeight,
		LocalTimeoutStamp: msg.LocalTimeoutStamp,
		QueryHeight: msg.QueryHeight,
		ClientId: msg.ClientId,
		Sender: msg.Sender,
	}

	k.SetSubmitCrossChainQuery(ctx,query)

	//TODO
	// Add Capability Key
	// capKey, err := k.scopedKeeper.NewCapability(ctx,msg.Id)
	// if err != nil {
	// 	return nil, sdkerrors.Wrapf(err, "could not create query capability for query ID %s ", msg.Id)
	// }
	

	// Log the query request
	k.Logger(ctx).Info("query sent","query_id", msg.GetQueryId())

	// emit event 
	EmitQueryEvent(ctx, msg)

	return &types.MsgSubmitCrossChainQueryResponse{Query: query}, nil
}

// SubmitCrossChainQueryResult Handling SubmitCrossChainQueryResult transaction
func (k Keeper) SubmitCrossChainQueryResult(goCtx context.Context, msg *types.MsgSubmitCrossChainQueryResult) (*types.MsgSubmitCrossChainQueryResultResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// check CrossChainQuery exist
	if _, found := k.GetSubmitCrossChainQuery(ctx, msg.Id); !found {
		return nil, types.ErrCrossChainQueryNotFound
	}

	// remove query from privateStore
	k.DeleteSubmitCrossChainQuery(ctx, msg.Id)

	queryResult := types.MsgSubmitCrossChainQueryResult{
		Id:     msg.Id,
		Result: msg.Result,
		Data:   msg.Data,
	}

	// store result in privateStore
	k.SetSubmitCrossChainQueryResult(ctx, queryResult)

	return &types.MsgSubmitCrossChainQueryResultResponse{}, nil
}
