package keeper

import (
	"context"
	"fmt"
	"slices"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/api/cometbft/abci/v1"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

var _ types.MsgServer = (*msgServer)(nil)

type msgServer struct {
	*Keeper
}

// NewMsgServerImpl returns an implementation of the ICS27 host MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// ModuleQuerySafe routes the queries to the keeper's query router if they are module_query_safe.
// This handler doesn't use the signer.
func (m msgServer) ModuleQuerySafe(goCtx context.Context, msg *types.MsgModuleQuerySafe) (*types.MsgModuleQuerySafeResponse, error) {

	responses := make([][]byte, len(msg.Requests))
	for i, query := range msg.Requests {
		isModuleQuerySafe := slices.Contains(m.mqsAllowList, query.Path)
		if !isModuleQuerySafe {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "not module query safe: %s", query.Path)
		}

		route := m.queryRouter.Route(query.Path)
		if route == nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "no route to query: %s", query.Path)
		}

		ctx := sdk.UnwrapSDKContext(goCtx)
		res, err := m.QueryRouterService.Invoke(ctx, &abci.QueryRequest{
			Path: query.Path,
			Data: query.Data,
		})
		if err != nil {
			m.Logger.Debug("query failed", "path", query.Path, "error", err)
			return nil, err
		}

		resp, ok := res.(*abci.QueryResponse)
		if !ok {
			return nil, fmt.Errorf("unexpected response type: %T", resp)
		}

		if resp == nil || resp.Value == nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "no response for query: %s", query.Path)
		}

		responses[i] = resp.Value
	}
	height := m.HeaderService.HeaderInfo(goCtx).Height

	return &types.MsgModuleQuerySafeResponse{Responses: responses, Height: uint64(height)}, nil
}

// UpdateParams updates the host submodule's params.
func (m msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if m.GetAuthority() != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", m.GetAuthority(), msg.Signer)
	}

	m.SetParams(ctx, msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
