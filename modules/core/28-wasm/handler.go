package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/ibc-go/modules/core/28-wasm/keeper"
	"github.com/cosmos/ibc-go/modules/core/28-wasm/types"
)

func HandleMsgPushNewWASMCode(ctx sdk.Context, k keeper.Keeper, msg *types.MsgPushNewWASMCode) (*types.MsgPushNewWASMCodeResponse, error) {
	if codeId, codeHash, err := k.PushNewWASMCode(ctx, msg.ClientType, msg.Code); err != nil {
		return nil, sdkerrors.Wrap(err, "pushing new wasm code failed")
	} else {
		return &types.MsgPushNewWASMCodeResponse{
			CodeId: codeId,
			CodeHash: codeHash,
		}, nil
	}
}
