package keeper

import (
	"context"
	"slices"
	"strings"

	gogoproto "github.com/cosmos/gogoproto/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	errorsmod "cosmossdk.io/errors"

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
func (m msgServer) ModuleQuerySafe(ctx context.Context, msg *types.MsgModuleQuerySafe) (*types.MsgModuleQuerySafeResponse, error) {
	responses := make([][]byte, len(msg.Requests))
	for i, query := range msg.Requests {
		isModuleQuerySafe := slices.Contains(m.mqsAllowList, query.Path)
		if !isModuleQuerySafe {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "not module query safe: %s", query.Path)
		}

		path := strings.TrimPrefix(query.Path, "/")
		pathFullName := protoreflect.FullName(strings.ReplaceAll(path, "/", "."))

		desc, err := gogoproto.GogoResolver.FindDescriptorByName(pathFullName)
		if err != nil {
			return nil, err
		}

		md, isGRPC := desc.(protoreflect.MethodDescriptor)
		if !isGRPC {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "no descriptor found for query path: %s", string(desc.FullName()))
		}

		msg := dynamicpb.NewMessage(md.Input())
		if err := m.cdc.Unmarshal(query.Data, msg); err != nil {
			return nil, err
		}

		res, err := m.QueryRouterService.Invoke(ctx, msg)
		if err != nil {
			m.Logger.Debug("query failed", "path", query.Path, "error", err)
			return nil, err
		}
		if res == nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "no response for query: %s", query.Path)
		}

		responses[i] = m.cdc.MustMarshal(res)
	}

	height := m.HeaderService.HeaderInfo(ctx).Height
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
