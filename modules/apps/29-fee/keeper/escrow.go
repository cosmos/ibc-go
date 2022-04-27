package keeper

import (
	"bytes"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

// escrowPacketFee sends the packet fee to the 29-fee module account to hold in escrow
func (k Keeper) escrowPacketFee(ctx sdk.Context, packetID channeltypes.PacketId, packetFee types.PacketFee) error {
	// check if the refund address is valid
	refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
	if err != nil {
		return err
	}

	refundAcc := k.authKeeper.GetAccount(ctx, refundAddr)
	if refundAcc == nil {
		return sdkerrors.Wrapf(types.ErrRefundAccNotFound, "account with address: %s not found", packetFee.RefundAddress)
	}

	coins := packetFee.Fee.Total()
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, refundAddr, types.ModuleName, coins); err != nil {
		return err
	}

	// multiple fees may be escrowed for a single packet, firstly create a slice containing the new fee
	// retrieve any previous fees stored in escrow for the packet and append them to the list
	fees := []types.PacketFee{packetFee}
	if feesInEscrow, found := k.GetFeesInEscrow(ctx, packetID); found {
		fees = append(fees, feesInEscrow.PacketFees...)
	}

	packetFees := types.NewPacketFees(fees)
	k.SetFeesInEscrow(ctx, packetID, packetFees)

	EmitIncentivizedPacket(ctx, packetID, packetFee)

	return nil
}

// DistributePacketFeesOnAcknowledgement pays all the acknowledgement & receive fees for a given packetID while refunding the timeout fees to the refund account.
func (k Keeper) DistributePacketFeesOnAcknowledgement(ctx sdk.Context, forwardRelayer string, reverseRelayer sdk.AccAddress, packetFees []types.PacketFee) {
	// cache context before trying to distribute fees
	// if the escrow account has insufficient balance then we want to avoid partially distributing fees
	cacheCtx, writeFn := ctx.CacheContext()

	forwardAddr, _ := sdk.AccAddressFromBech32(forwardRelayer)

	for _, packetFee := range packetFees {
		if !k.EscrowAccountHasBalance(cacheCtx, packetFee.Fee.Total()) {
			// if the escrow account does not have sufficient funds then there must exist a severe bug
			// the fee module should be locked until manual intervention fixes the issue
			// a locked fee module will simply skip fee logic, all channels will temporarily function as
			// fee disabled channels
			// NOTE: we use the uncached context to lock the fee module so that the state changes from
			// locking the fee module are persisted
			k.lockFeeModule(ctx)

			return
		}

		// check if refundAcc address works
		refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
		if err != nil {
			panic(fmt.Sprintf("could not parse refundAcc %s to sdk.AccAddress", packetFee.RefundAddress))
		}

		k.distributePacketFeeOnAcknowledgement(cacheCtx, refundAddr, forwardAddr, reverseRelayer, packetFee)
	}

	// write the cache
	writeFn()

	// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
	ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
}

// distributePacketFeeOnAcknowledgement pays the receive fee for a given packetID while refunding the timeout fee to the refund account associated with the Fee.
// If there was no forward relayer or the associated forward relayer address is blocked, the receive fee is refunded.
func (k Keeper) distributePacketFeeOnAcknowledgement(ctx sdk.Context, refundAddr, forwardRelayer, reverseRelayer sdk.AccAddress, packetFee types.PacketFee) {
	// distribute fee to valid forward relayer address otherwise refund the fee
	if !forwardRelayer.Empty() && !k.bankKeeper.BlockedAddr(forwardRelayer) {
		// distribute fee for forward relaying
		k.distributeFee(ctx, forwardRelayer, refundAddr, packetFee.Fee.RecvFee)
	} else {
		// refund onRecv fee as forward relayer is not valid address
		k.distributeFee(ctx, refundAddr, refundAddr, packetFee.Fee.RecvFee)
	}

	// distribute fee for reverse relaying
	k.distributeFee(ctx, reverseRelayer, refundAddr, packetFee.Fee.AckFee)

	// refund timeout fee for unused timeout
	k.distributeFee(ctx, refundAddr, refundAddr, packetFee.Fee.TimeoutFee)

}

// DistributePacketsFeesOnTimeout pays all the timeout fees for a given packetID while refunding the acknowledgement & receive fees to the refund account.
func (k Keeper) DistributePacketFeesOnTimeout(ctx sdk.Context, timeoutRelayer sdk.AccAddress, packetFees []types.PacketFee) {
	// cache context before trying to distribute fees
	// if the escrow account has insufficient balance then we want to avoid partially distributing fees
	cacheCtx, writeFn := ctx.CacheContext()

	for _, packetFee := range packetFees {
		if !k.EscrowAccountHasBalance(cacheCtx, packetFee.Fee.Total()) {
			// if the escrow account does not have sufficient funds then there must exist a severe bug
			// the fee module should be locked until manual intervention fixes the issue
			// a locked fee module will simply skip fee logic, all channels will temporarily function as
			// fee disabled channels
			// NOTE: we use the uncached context to lock the fee module so that the state changes from
			// locking the fee module are persisted
			k.lockFeeModule(ctx)

			return
		}

		// check if refundAcc address works
		refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
		if err != nil {
			panic(fmt.Sprintf("could not parse refundAcc %s to sdk.AccAddress", packetFee.RefundAddress))
		}

		k.distributePacketFeeOnTimeout(cacheCtx, refundAddr, timeoutRelayer, packetFee)
	}

	// write the cache
	writeFn()

	// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
	ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
}

// distributePacketFeeOnTimeout pays the timeout fee to the timeout relayer and refunds the acknowledgement & receive fee.
func (k Keeper) distributePacketFeeOnTimeout(ctx sdk.Context, refundAddr, timeoutRelayer sdk.AccAddress, packetFee types.PacketFee) {
	// refund receive fee for unused forward relaying
	k.distributeFee(ctx, refundAddr, refundAddr, packetFee.Fee.RecvFee)

	// refund ack fee for unused reverse relaying
	k.distributeFee(ctx, refundAddr, refundAddr, packetFee.Fee.AckFee)

	// distribute fee for timeout relaying
	k.distributeFee(ctx, timeoutRelayer, refundAddr, packetFee.Fee.TimeoutFee)
}

// distributeFee will attempt to distribute the escrowed fee to the receiver address.
// If the distribution fails for any reason (such as the receiving address being blocked),
// the state changes will be discarded.
func (k Keeper) distributeFee(ctx sdk.Context, receiver, refundAccAddress sdk.AccAddress, fee sdk.Coins) {
	// cache context before trying to distribute fees
	cacheCtx, writeFn := ctx.CacheContext()

	err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, receiver, fee)
	if err != nil {
		if bytes.Equal(receiver, refundAccAddress) {
			return // if sending to the refund address already failed, then return (no-op)
		}

		// if an error is returned from x/bank and the receiver is not the refundAccAddress
		// then attempt to refund the fee to the original sender
		err := k.bankKeeper.SendCoinsFromModuleToAccount(cacheCtx, types.ModuleName, refundAccAddress, fee)
		if err != nil {
			return // if sending to the refund address fails, no-op
		}
	}

	// write the cache
	writeFn()

	// NOTE: The context returned by CacheContext() refers to a new EventManager, so it needs to explicitly set events to the original context.
	ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
}

// RefundFeesOnChannelClosure will refund all fees associated with the given port and channel identifiers.
// If the escrow account runs out of balance then fee module will become locked as this implies the presence
// of a severe bug. When the fee module is locked, no fee distributions will be performed.
// Please see ADR 004 for more information.
func (k Keeper) RefundFeesOnChannelClosure(ctx sdk.Context, portID, channelID string) error {
	identifiedPacketFees := k.GetIdentifiedPacketFeesForChannel(ctx, portID, channelID)

	// cache context before trying to distribute fees
	// if the escrow account has insufficient balance then we want to avoid partially distributing fees
	cacheCtx, writeFn := ctx.CacheContext()

	for _, identifiedPacketFee := range identifiedPacketFees {
		for _, packetFee := range identifiedPacketFee.PacketFees {

			if !k.EscrowAccountHasBalance(cacheCtx, packetFee.Fee.Total()) {
				// if the escrow account does not have sufficient funds then there must exist a severe bug
				// the fee module should be locked until manual intervention fixes the issue
				// a locked fee module will simply skip fee logic, all channels will temporarily function as
				// fee disabled channels
				// NOTE: we use the uncached context to lock the fee module so that the state changes from
				// locking the fee module are persisted
				k.lockFeeModule(ctx)

				// return a nil error so state changes are committed but distribution stops
				return nil
			}

			refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
			if err != nil {
				return err
			}

			// if the refund address is blocked, skip and continue distribution
			if k.bankKeeper.BlockedAddr(refundAddr) {
				continue
			}

			// refund all fees to refund address
			// Use SendCoins rather than the module account send functions since refund address may be a user account or module address.
			moduleAcc := k.GetFeeModuleAddress()
			if err = k.bankKeeper.SendCoins(cacheCtx, moduleAcc, refundAddr, packetFee.Fee.Total()); err != nil {
				return err
			}

		}

		k.DeleteFeesInEscrow(cacheCtx, identifiedPacketFee.PacketId)
	}

	// write the cache
	writeFn()

	return nil
}
