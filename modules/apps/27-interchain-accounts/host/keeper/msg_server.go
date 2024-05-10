package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	_ "cosmossdk.io/api/cosmos/bank/v1beta1"    // workaround to successfully retrieve bank module safe queries
	_ "cosmossdk.io/api/cosmos/staking/v1beta1" // workaround to successfully retrieve staking module safe queries
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
)

var _ types.MsgServer = (*msgServer)(nil)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the ICS27 host MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	if keeper.queryRouter == nil {
		panic("query router must not be nil")
	}
	return &msgServer{Keeper: keeper}
}

// ModuleQuerySafe routes the queries to the keeper's query router if they are module_query_safe.
// This handler doesn't use the signer.
func (m msgServer) ModuleQuerySafe(goCtx context.Context, msg *types.MsgModuleQuerySafe) (*types.MsgModuleQuerySafeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	responses := make([][]byte, len(msg.Requests))
	for i, query := range msg.Requests {
		var isModuleQuerySafe bool
		for _, allowedQueryPath := range m.mqsAllowList {
			if allowedQueryPath == query.Path {
				isModuleQuerySafe = true
				break
			}
		}
		if !isModuleQuerySafe {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "not module query safe: %s", query.Path)
		}

		route := m.queryRouter.Route(query.Path)
		if route == nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "no route to query: %s", query.Path)
		}

		res, err := route(ctx, abci.RequestQuery{
			Path: query.Path,
			Data: query.Data,
		})
		if err != nil {
			m.Logger(ctx).Debug("query failed", "path", query.Path, "error", err)
			return nil, err
		}
		if res.Value == nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "no response for query: %s", query.Path)
		}

		responses[i] = res.Value
	}

	return &types.MsgModuleQuerySafeResponse{Responses: responses, Height: uint64(ctx.BlockHeight())}, nil
}
