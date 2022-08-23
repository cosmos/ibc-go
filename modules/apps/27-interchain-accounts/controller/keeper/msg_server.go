package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
)

var _ types.MsgServer = Keeper{}

// RegisterAccount defines a rpc handler for MsgRegisterAccount
func (k Keeper) RegisterAccount(goCtx context.Context, msg *types.MsgRegisterAccount) (*types.MsgRegisterAccountResponse, error) {
	return &types.MsgRegisterAccountResponse{}, nil
}

// SubmitTx defines a rpc handler for MsgSendTx
func (k Keeper) SubmitTx(goCtx context.Context, msg *types.MsgSubmitTx) (*types.MsgSubmitTxResponse, error) {
	return &types.MsgSubmitTxResponse{}, nil
}
