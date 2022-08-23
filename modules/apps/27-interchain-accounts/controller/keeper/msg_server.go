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
