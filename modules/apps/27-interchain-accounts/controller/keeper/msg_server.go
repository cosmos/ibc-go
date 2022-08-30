package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
)

var _ types.MsgServer = Keeper{}

// RegisterAccount defines a rpc handler for MsgRegisterAccount
func (k Keeper) RegisterAccount(goCtx context.Context, msg *types.MsgRegisterAccount) (*types.MsgRegisterAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	channelID, err := k.registerInterchainAccount(ctx, msg.ConnectionId, msg.Owner, msg.Version)
	if err != nil {
		return nil, err
	}

	return &types.MsgRegisterAccountResponse{
		ChannelId: channelID,
	}, nil
}

// SubmitTx defines a rpc handler for MsgSubmitTx
func (k Keeper) SubmitTx(goCtx context.Context, msg *types.MsgSubmitTx) (*types.MsgSubmitTxResponse, error) {
	return &types.MsgSubmitTxResponse{}, nil
}
