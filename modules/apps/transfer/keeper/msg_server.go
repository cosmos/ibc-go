package keeper

import (
	"context"
	"slices"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
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
	if err := k.bankKeeper.IsSendEnabledCoins(ctx, coins...); err != nil {
		return nil, errorsmod.Wrapf(types.ErrSendDisabled, err.Error())
	}

	if k.isBlockedAddr(sender) {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to send funds", sender)
	}

	appVersion, found := k.ics4Wrapper.GetAppVersion(ctx, msg.SourcePort, msg.SourceChannel)
	if !found {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "application version not found for source port: %s and source channel: %s", msg.SourcePort, msg.SourceChannel)
	}

	if appVersion == types.V1 {
		// ics20-1 only supports a single coin, so if that is the current version, we must only process a single coin.
		if len(msg.Tokens) > 1 {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot transfer multiple coins with %s", types.V1)
		}

		// ics20-1 does not support forwarding, so if that is the current version, we must reject the transfer.
		if msg.HasForwarding() {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot forward coins with %s", types.V1)
		}
	}

	if msg.Forwarding.GetUnwind() {
		msg, err = k.unwindHops(ctx, msg)
		if err != nil {
			return nil, err
		}
	}

	tokens := make([]types.Token, 0, len(coins))
	for _, coin := range coins {
		token, err := k.tokenFromCoin(ctx, coin)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, token)
	}

	packetDataBz, err := createPacketDataBytesFromVersion(appVersion, sender.String(), msg.Receiver, msg.Memo, tokens, msg.Forwarding.GetHops())
	if err != nil {
		return nil, err
	}

	// packetData := types.NewFungibleTokenPacketData(
	// 	fullDenomPath, msg.Token.Amount.String(), sender.String(), msg.Receiver, msg.Memo,
	// )

	msgSendPacket := &channeltypes.MsgSendPacket{
		PortId:           msg.SourcePort,
		ChannelId:        msg.SourceChannel,
		TimeoutHeight:    msg.TimeoutHeight,
		TimeoutTimestamp: msg.TimeoutTimestamp,
		Data:             packetDataBz,
		Signer:           msg.Sender,
	}

	handler := k.msgRouter.Handler(msgSendPacket)
	res, err := handler(ctx, msgSendPacket)
	if err != nil {
		return nil, err
	}

	sendPacketResp, ok := res.MsgResponses[0].GetCachedValue().(*channeltypes.MsgSendPacketResponse)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to convert %T message response to %T", res.MsgResponses[0].GetCachedValue(), &channeltypes.MsgSendPacketResponse{})
	}

	// NOTE: The sdk msg handler creates a new EventManager, so events must be correctly propagated back to the current context
	ctx.EventManager().EmitEvents(res.GetEvents())

	k.Logger(ctx).Info("IBC fungible token transfer", "token", msg.Token.Denom, "amount", msg.Token.Amount.String(), "sender", msg.Sender, "receiver", msg.Receiver)

	// ctx.EventManager().EmitEvents(sdk.Events{
	// 	sdk.NewEvent(
	// 		types.EventTypeTransfer,
	// 		sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
	// 		sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
	// 		sdk.NewAttribute(types.AttributeKeyAmount, msg.Token.Amount.String()),
	// 		sdk.NewAttribute(types.AttributeKeyDenom, msg.Token.Denom),
	// 		sdk.NewAttribute(types.AttributeKeyMemo, msg.Memo),
	// 	),
	// 	sdk.NewEvent(
	// 		sdk.EventTypeMessage,
	// 		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	// 	),
	// })

	return &types.MsgTransferResponse{Sequence: sendPacketResp.Sequence}, nil
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

	// Message is validate again, this would only fail if hops now exceeds maximum allowed.
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
