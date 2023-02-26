package keeper

import (
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v7/modules/light-clients/08-wasm/types"
)

var _ types.MsgServer = Keeper{}

// PushNewWasmCode defines a rpc handler method for MsgPushNewWasmCode
func (k Keeper) PushNewWasmCode(goCtx context.Context, msg *types.MsgPushNewWasmCode) (*types.MsgPushNewWasmCodeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.authority != msg.Signer {
		return nil, sdkerrors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Signer)
	}

	codeId, err := k.storeWasmCode(ctx, msg.Code)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "pushing new wasm code failed")
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			clienttypes.EventTypePushWasmCode,
			sdk.NewAttribute(clienttypes.AttributeKeyWasmCodeID, hex.EncodeToString(codeId)),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, clienttypes.AttributeValueCategory),
		),
	})

	return &types.MsgPushNewWasmCodeResponse{
		CodeId: codeId,
	}, nil
}
