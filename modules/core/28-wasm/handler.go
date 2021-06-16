package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/modules/core/28-wasm/keeper"
	"github.com/cosmos/ibc-go/modules/core/28-wasm/types"
)

func HandleMsgPushNewWASMCode(ctx sdk.Context, k keeper.Keeper, msg *types.MsgPushNewWASMCode) (*types.MsgPushNewWASMCodeResponse, error) {
	codeID, err := k.PushNewWASMCode(ctx, msg.Code)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "pushing new wasm code failed")
	}

	return &types.MsgPushNewWASMCodeResponse{
		CodeId: codeID,
	}, nil
}
