package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/cosmos/ibc-go/v7/internal/errors"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

var _ types.MsgServer = Keeper{}

// Transfer defines an rpc handler method for MsgTransfer.
func (k Keeper) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.GetSendEnabled(ctx) {
		return nil, types.ErrSendDisabled
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	if !k.bankKeeper.IsSendEnabledCoin(ctx, msg.Token) {
		return nil, errorsmod.Wrapf(types.ErrSendDisabled, "%s transfers are currently disabled", msg.Token.Denom)
	}

	if k.bankKeeper.BlockedAddr(sender) {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to send funds", sender)
	}

	// Use a cached context to prevent accidental state changes when setting the total amount in escrow
	cacheCtx, writeFn := ctx.CacheContext()
	sequence, err := k.sendTransfer(
		cacheCtx, msg.SourcePort, msg.SourceChannel, msg.Token, sender, msg.Receiver, msg.TimeoutHeight, msg.TimeoutTimestamp,
		msg.Memo)
	if err != nil {
		return nil, err
	}
	writeFn()

	k.Logger(ctx).Info("IBC fungible token transfer", "token", msg.Token.Denom, "amount", msg.Token.Amount.String(), "sender", msg.Sender, "receiver", msg.Receiver)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTransfer,
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
			sdk.NewAttribute(types.AttributeKeyAmount, msg.Token.Amount.String()),
			sdk.NewAttribute(types.AttributeKeyDenom, msg.Token.Denom),
			sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
	})

	return &types.MsgTransferResponse{Sequence: sequence}, nil
}
