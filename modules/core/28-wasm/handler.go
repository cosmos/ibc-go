package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/keeper"
	"github.com/cosmos/ibc-go/v5/modules/core/28-wasm/types"
)

func HandleMsgPushNewWasmCode(ctx sdk.Context, k keeper.Keeper, msg *types.MsgPushNewWasmCode) (*types.MsgPushNewWasmCodeResponse, error) {
	codeID, err := k.PushNewWasmCode(ctx, msg.Code)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "pushing new wasm code failed")
	}

	return &types.MsgPushNewWasmCodeResponse{
		CodeId: codeID,
	}, nil
}
