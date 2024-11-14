package keeper

import (
	"context"
	"slices"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
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

	coins := msg.GetCoins()

	if err := k.BankKeeper.IsSendEnabledCoins(ctx, coins...); err != nil {
		return nil, errorsmod.Wrapf(types.ErrSendDisabled, err.Error())
	}

	if k.IsBlockedAddr(sender) {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to send funds", sender)
	}

	if msg.Forwarding.GetUnwind() {
		msg, err = k.unwindHops(ctx, msg)
		if err != nil {
			return nil, err
		}
	}

	sequence, err := k.sendTransfer(
		ctx, msg.SourcePort, msg.SourceChannel, coins, sender, msg.Receiver, msg.TimeoutHeight, msg.TimeoutTimestamp,
		msg.Memo, msg.Forwarding.GetHops())
	if err != nil {
		return nil, err
	}

	k.Logger(ctx).Info("IBC fungible token transfer", "tokens", coins, "sender", msg.Sender, "receiver", msg.Receiver)

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

// unwindHops unwinds the hops present in the tokens denomination and returns the message modified to reflect
// the unwound path to take. It assumes that only a single token is present (as this is verified in ValidateBasic)
// in the tokens list and ensures that the token is not native to the chain.
func (k Keeper) unwindHops(ctx sdk.Context, msg *types.MsgTransfer) (*types.MsgTransfer, error) {
	unwindHops, err := k.getUnwindHops(ctx, msg.GetCoins())
	if err != nil {
		return nil, err
	}

	// Update message fields.
	msg.SourcePort, msg.SourceChannel = unwindHops[0].PortId, unwindHops[0].ChannelId
	msg.Forwarding.Hops = append(unwindHops[1:], msg.Forwarding.Hops...)
	msg.Forwarding.Unwind = false

	// Message is validated again, this would only fail if hops now exceeds maximum allowed.
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	return msg, nil
}

// getUnwindHops returns the hops to be used during unwinding. If coins consists of more than
// one coin, all coins must have the exact same trace, else an error is returned. getUnwindHops
// also validates that the coins are not native to the chain.
func (k Keeper) getUnwindHops(ctx sdk.Context, coins sdk.Coins) ([]types.Hop, error) {
	// Sanity: validation for MsgTransfer ensures coins are not empty.
	if len(coins) == 0 {
		return nil, errorsmod.Wrap(types.ErrInvalidForwarding, "coins cannot be empty")
	}

	token, err := k.tokenFromCoin(ctx, coins[0])
	if err != nil {
		return nil, err
	}

	if token.Denom.IsNative() {
		return nil, errorsmod.Wrap(types.ErrInvalidForwarding, "cannot unwind a native token")
	}

	unwindTrace := token.Denom.Trace
	for _, coin := range coins[1:] {
		token, err := k.tokenFromCoin(ctx, coin)
		if err != nil {
			return nil, err
		}

		// Implicitly ensures coin we're iterating over is not native.
		if !slices.Equal(token.Denom.Trace, unwindTrace) {
			return nil, errorsmod.Wrap(types.ErrInvalidForwarding, "cannot unwind tokens with different traces.")
		}
	}

	return unwindTrace, nil
}
