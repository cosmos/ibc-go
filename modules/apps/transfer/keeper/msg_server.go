package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

var _ types.MsgServer = (*Keeper)(nil)

// Transfer defines an rpc handler method for MsgTransfer.
func (k Keeper) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.GetParams(ctx).SendEnabled {
		return nil, types.ErrSendDisabled
	}

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	// tokens will always be an array, but may contain a single element
	// if the ics20-1 token field is populated, and the ics20-2 array is empty.
	tokens := msg.GetTokens()

	appVersion, found := k.ics4Wrapper.GetAppVersion(ctx, msg.SourcePort, msg.SourceChannel)
	if !found {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "application version not found for source port: %s and source channel: %s", msg.SourcePort, msg.SourceChannel)
	}

	// ics20-1 only supports a single token, so if that is the current version, we must only process a single token.
	if appVersion == types.ICS20V1 && len(tokens) > 1 {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot transfer multiple tokens with ics20-1")
	}

	for _, token := range tokens {
		if !k.bankKeeper.IsSendEnabledCoin(ctx, token) {
			return nil, errorsmod.Wrapf(types.ErrSendDisabled, "transfers are currently disabled for %s", token.Denom)
		}
	}

	if k.bankKeeper.BlockedAddr(sender) {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to send funds", sender)
	}

	sequence, err := k.sendTransfer(
		ctx, msg.SourcePort, msg.SourceChannel, tokens, sender, msg.Receiver, msg.TimeoutHeight, msg.TimeoutTimestamp,
		msg.Memo)
	if err != nil {
		return nil, err
	}

	for _, token := range tokens {
		k.Logger(ctx).Info("IBC fungible token transfer", "token", token.Denom, "amount", token.Amount.String(), "sender", msg.Sender, "receiver", msg.Receiver)
		ctx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent(types.EventTypeTransfer,
				sdk.NewAttribute(types.AttributeKeySender, msg.Sender),
				sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
				sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
				sdk.NewAttribute(types.AttributeKeyDenom, token.Denom),
				sdk.NewAttribute(types.AttributeKeyAmount, token.Amount.String()),
			),
			sdk.NewEvent(
				sdk.EventTypeMessage,
				sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			),
		})
	}

	return &types.MsgTransferResponse{Sequence: sequence}, nil
}

// UpdateParams defines an rpc handler method for MsgUpdateParams. Updates the ibc-transfer module's parameters.
func (k Keeper) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.GetAuthority() != msg.Signer {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s, got %s", k.GetAuthority(), msg.Signer)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.SetParams(ctx, msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
