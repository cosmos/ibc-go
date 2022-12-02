package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func HandleMsgPushNewWasmCode(ctx sdk.Context, k Keeper, msg *MsgPushNewWasmCode) (*MsgPushNewWasmCodeResponse, error) {
	codeID, err := k.PushNewWasmCode(ctx, msg.Code)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "pushing new wasm code failed")
	}

	return &MsgPushNewWasmCodeResponse{
		CodeId: codeID,
	}, nil
}