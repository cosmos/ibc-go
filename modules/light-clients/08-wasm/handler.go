package wasm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func HandleMsgPushNewWasmCode(ctx sdk.Context, k Keeper, msg *MsgPushNewWasmCode) (*MsgPushNewWasmCodeResponse, error) {
	if (k.authority != msg.Signer) {
		return nil, sdkerrors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Signer)
	}

	codeID, err := k.PushNewWasmCode(ctx, msg.Code)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "pushing new wasm code failed")
	}

	return &MsgPushNewWasmCodeResponse{
		CodeId: codeID,
	}, nil
}