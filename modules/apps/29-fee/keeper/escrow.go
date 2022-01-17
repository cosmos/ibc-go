package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
)

// EscrowPacketFee sends the packet fee to the 29-fee module account to hold in escrow
func (k Keeper) EscrowPacketFee(ctx sdk.Context, identifiedFee *types.IdentifiedPacketFee) error {
	if !k.IsFeeEnabled(ctx, identifiedFee.PacketId.PortId, identifiedFee.PacketId.ChannelId) {
		// users may not escrow fees on this channel. Must send packets without a fee message
		return sdkerrors.Wrap(types.ErrFeeNotEnabled, "cannot escrow fee for packet")
	}
	// check if the refund account exists
	refundAcc, err := sdk.AccAddressFromBech32(identifiedFee.RefundAddress)
	if err != nil {
		return err
	}

	hasRefundAcc := k.authKeeper.GetAccount(ctx, refundAcc)
	if hasRefundAcc == nil {
		return sdkerrors.Wrap(types.ErrRefundAccNotFound, fmt.Sprintf("account with address: %s not found", refundAcc))
	}

	coins := identifiedFee.Fee.ReceiveFee
	coins = coins.Add(identifiedFee.Fee.AckFee...)
	coins = coins.Add(identifiedFee.Fee.TimeoutFee...)

	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, refundAcc, types.ModuleName, coins,
	); err != nil {
		return err
	}

	// Store fee in state for reference later
	k.SetFeeInEscrow(ctx, identifiedFee)
	return nil
}

// DistributePacketFees pays the acknowledgement fee & receive fee for a given packetId while refunding the timeout fee to the refund account associated with the Fee.
func (k Keeper) DistributePacketFees(ctx sdk.Context, refundAcc, forwardRelayer string, reverseRelayer sdk.AccAddress, feeInEscrow types.IdentifiedPacketFee) {
	// distribute fee for forward relaying
	forward, err := sdk.AccAddressFromBech32(forwardRelayer)
	if err == nil {
		k.distributeFee(ctx, forward, feeInEscrow.Fee.ReceiveFee)
	}

	// distribute fee for reverse relaying
	k.distributeFee(ctx, reverseRelayer, feeInEscrow.Fee.AckFee)

	// refund timeout fee refund,
	refundAddr, err := sdk.AccAddressFromBech32(refundAcc)
	if err != nil {
		panic(fmt.Sprintf("could not parse refundAcc %s to sdk.AccAddress", refundAcc))
	}

	// refund timeout fee for unused timeout
	k.distributeFee(ctx, refundAddr, feeInEscrow.Fee.TimeoutFee)

	// removes the fee from the store as fee is now paid
	k.DeleteFeeInEscrow(ctx, feeInEscrow.PacketId)
}

// DistributePacketsFeesTimeout pays the timeout fee for a given packetId while refunding the acknowledgement fee & receive fee to the refund account associated with the Fee
func (k Keeper) DistributePacketFeesOnTimeout(ctx sdk.Context, refundAcc string, timeoutRelayer sdk.AccAddress, feeInEscrow types.IdentifiedPacketFee) {
	// check if refundAcc address works
	refundAddr, err := sdk.AccAddressFromBech32(refundAcc)
	if err != nil {
		panic(fmt.Sprintf("could not parse refundAcc %s to sdk.AccAddress", refundAcc))
	}

	// refund receive fee for unused forward relaying
	k.distributeFee(ctx, refundAddr, feeInEscrow.Fee.ReceiveFee)

	// refund ack fee for unused reverse relaying
	k.distributeFee(ctx, refundAddr, feeInEscrow.Fee.AckFee)

	// distribute fee for timeout relaying
	k.distributeFee(ctx, timeoutRelayer, feeInEscrow.Fee.TimeoutFee)

	// removes the fee from the store as fee is now paid
	k.DeleteFeeInEscrow(ctx, feeInEscrow.PacketId)
}

// distributeFee will attempt to distribute the escrowed fee to the receiver address.
// If the distribution fails for any reason (such as the receiving address being blocked),
// the state changes will be discarded.
func (k Keeper) distributeFee(ctx sdk.Context, receiver sdk.AccAddress, fee sdk.Coins) {
	// cache context before trying to send to reverse relayer
	cacheCtx, writeFn := ctx.CacheContext()

	err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, receiver, fee)
	if err == nil {
		// write the cache
		writeFn()

		// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
		ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
	}
}

func (k Keeper) RefundFeesOnChannel(ctx sdk.Context, portID, channelID string) error {

	var refundErr error

	k.IterateChannelFeesInEscrow(ctx, portID, channelID, func(identifiedFee types.IdentifiedPacketFee) (stop bool) {
		refundAccAddr, err := sdk.AccAddressFromBech32(identifiedFee.RefundAddress)
		if err != nil {
			refundErr = err
			return true
		}

		// refund all fees to refund address
		// Use SendCoins rather than the module account send functions since refund address may be a user account or module address.
		// if any `SendCoins` call returns an error, we return error and stop iteration
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, refundAccAddr, identifiedFee.Fee.ReceiveFee)
		if err != nil {
			refundErr = err
			return true
		}

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, refundAccAddr, identifiedFee.Fee.AckFee)
		if err != nil {
			refundErr = err
			return true
		}
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, refundAccAddr, identifiedFee.Fee.TimeoutFee)
		if err != nil {
			refundErr = err
			return true
		}
		return false
	})

	return refundErr
}
