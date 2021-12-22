package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
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
		return sdkerrors.Wrap(types.ErrRefundAccNotFound, fmt.Sprintf("Account with address: %s not found", refundAcc))
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
	// feeInEscrow/<port-id>/<channel-id>/packet/<sequence-id>/ -> Fee (timeout, ack, onrecv)
	k.SetFeeInEscrow(ctx, identifiedFee)
	return nil
}

// DistributeFee pays the acknowledgement fee & receive fee for a given packetId while refunding the timeout fee to the refund account associated with the Fee
func (k Keeper) DistributeFee(ctx sdk.Context, refundAcc, forwardRelayer, reverseRelayer string, packetID *channeltypes.PacketId) error {
	var packetIdErr error
	var distributeFwdErr error
	var distributeReverseErr error
	var refundAccErr error

	// check if there is a Fee in escrow for the given packetId
	feeInEscrow, found := k.GetFeeInEscrow(ctx, packetID)
	if !found {
		packetIdErr = sdkerrors.Wrapf(types.ErrFeeNotFound, "with channelID %s, sequence %d", packetID.ChannelId, packetID.Sequence)
	}

	// cache context before trying to send to forward relayer
	cacheCtx, writeFn := ctx.CacheContext()

	fwd, err := sdk.AccAddressFromBech32(forwardRelayer)
	if err == nil {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, fwd, feeInEscrow.Fee.ReceiveFee)
		if err == nil {
			// write the cache
			writeFn()
			// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
			ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
		}
	} else {
		distributeFwdErr = sdkerrors.Wrap(err, "failed to send fee to forward relayer")
	}

	// cache context before trying to send to reverse relayer
	cacheCtx, writeFn = ctx.CacheContext()

	reverse, err := sdk.AccAddressFromBech32(reverseRelayer)
	if err == nil {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, reverse, feeInEscrow.Fee.AckFee)
		if err == nil {
			// write the cache
			writeFn()
			// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
			ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
		}
	} else {
		distributeReverseErr = sdkerrors.Wrap(err, "failed to send fee to reverse relayer")
	}

	// cache context before trying to send timeout fee to refund address
	cacheCtx, writeFn = ctx.CacheContext()

	refund, err := sdk.AccAddressFromBech32(refundAcc)
	if err == nil {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, refund, feeInEscrow.Fee.TimeoutFee)
		if err == nil {
			// write the cache
			writeFn()
			// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
			ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
		}
	} else {
		refundAccErr = sdkerrors.Wrap(err, "refunding timeout fee")
	}

	// removes the fee from the store as fee is now paid
	k.DeleteFeeInEscrow(ctx, packetID)

	// check if there are any errors, otherwise return nil
	if packetIdErr != nil || distributeFwdErr != nil || distributeReverseErr != nil || refundAccErr != nil {
		return sdkerrors.Wrapf(types.ErrFeeDistribution, "error in distributing fee: %s %s %s %s", packetIdErr, distributeFwdErr, distributeReverseErr, refundAccErr)
	}
	return nil
}

// DistributeFeeTimeout pays the timeout fee for a given packetId while refunding the acknowledgement fee & receive fee to the refund account associated with the Fee
func (k Keeper) DistributeFeeTimeout(ctx sdk.Context, refundAcc, timeoutRelayer string, packetID *channeltypes.PacketId) error {
	var packetIdErr error
	var refundAccErr error
	var distributeTimeoutErr error

	// check if there is a Fee in escrow for the given packetId
	feeInEscrow, found := k.GetFeeInEscrow(ctx, packetID)
	if !found {
		packetIdErr = sdkerrors.Wrapf(types.ErrFeeNotFound, "for packetID %s", packetID)
	}

	// cache context before trying to refund the receive fee
	cacheCtx, writeFn := ctx.CacheContext()

	// check if refundAcc address works
	refund, err := sdk.AccAddressFromBech32(refundAcc)
	if err == nil {
		// first try to refund receive fee
		err = k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, refund, feeInEscrow.Fee.ReceiveFee)
		if err == nil {
			// write the cache
			writeFn()
			// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
			ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
		} else {
			// set refundAccErr to error resulting from failed refund
			refundAccErr = sdkerrors.Wrap(err, "error refunding receive fee")
		}

		// then try to refund ack fee
		err = k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, refund, feeInEscrow.Fee.AckFee)
		if err == nil {
			// write the cache
			writeFn()
			// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
			ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
		} else {
			// set refundAccErr to error resulting from failed refund
			refundAccErr = sdkerrors.Wrap(err, "error refunding ack fee")
		}
	} else {
		refundAccErr = sdkerrors.Wrap(err, "failed to parse refund account address")
	}

	// parse timeout relayer address
	cacheCtx, writeFn = ctx.CacheContext()
	timeout, err := sdk.AccAddressFromBech32(timeoutRelayer)
	if err != nil {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, timeout, feeInEscrow.Fee.TimeoutFee)
		if err == nil {
			// write the cache
			writeFn()
			// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
			ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
		}
	} else {
		distributeTimeoutErr = sdkerrors.Wrap(err, "error sending fee to timeout relayer")
	}

	// removes the fee from the store as fee is now paid
	k.DeleteFeeInEscrow(ctx, packetID)

	// check if there are any errors, otherwise return nil
	if packetIdErr != nil || refundAccErr != nil || distributeTimeoutErr != nil {
		return sdkerrors.Wrapf(types.ErrFeeDistribution, "error in distributing fee: %s %s %s", packetIdErr, refundAccErr, distributeTimeoutErr)
	}
	return nil
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
