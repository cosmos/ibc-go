package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

// escrowPacketFee sends the packet fee to the 29-fee module account to hold in escrow
func (k Keeper) escrowPacketFee(ctx context.Context, packetID channeltypes.PacketId, packetFee types.PacketFee) error {
	// check if the refund address is valid
	refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
	if err != nil {
		return err
	}

	refundAcc := k.authKeeper.GetAccount(ctx, refundAddr)
	if refundAcc == nil {
		return errorsmod.Wrapf(types.ErrRefundAccNotFound, "account with address: %s not found", packetFee.RefundAddress)
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

	return k.emitIncentivizedPacketEvent(ctx, packetID, packetFees)
}

// DistributePacketFeesOnAcknowledgement pays all the acknowledgement & receive fees for a given packetID while refunding the timeout fees to the refund account.
func (k Keeper) DistributePacketFeesOnAcknowledgement(ctx context.Context, forwardRelayer string, reverseRelayer sdk.AccAddress, packetFees []types.PacketFee, packetID channeltypes.PacketId) {
	// use branched multistore for distribution of fees.
	// if the escrow account has insufficient balance then we want to avoid partially distributing fees
	if err := k.BranchService.Execute(ctx, func(ctx context.Context) error {
		// forward relayer address will be empty if conversion fails
		forwardAddr, _ := sdk.AccAddressFromBech32(forwardRelayer)

		for _, packetFee := range packetFees {
			if !k.EscrowAccountHasBalance(ctx, packetFee.Fee.Total()) {
				// NOTE: we lock the fee module on error return so that the state changes are persisted
				return ibcerrors.ErrInsufficientFunds
			}

			// check if refundAcc address works
			refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
			if err != nil {
				panic(fmt.Errorf("could not parse refundAcc %s to sdk.AccAddress", packetFee.RefundAddress))
			}

			k.distributePacketFeeOnAcknowledgement(ctx, refundAddr, forwardAddr, reverseRelayer, packetFee)
		}

		return nil
	}); err != nil {
		if errors.Is(err, ibcerrors.ErrInsufficientFunds) {
			// if the escrow account does not have sufficient funds then there must exist a severe bug
			// the fee module should be locked until manual intervention fixes the issue
			// a locked fee module will simply skip fee logic, all channels will temporarily function as
			// fee disabled channels
			k.lockFeeModule(ctx)
			return
		}
	}

	// removes the fees from the store as fees are now paid
	k.DeleteFeesInEscrow(ctx, packetID)
}

// distributePacketFeeOnAcknowledgement pays the receive fee for a given packetID while refunding the timeout fee to the refund account associated with the Fee.
// If there was no forward relayer or the associated forward relayer address is blocked, the receive fee is refunded.
func (k Keeper) distributePacketFeeOnAcknowledgement(ctx context.Context, refundAddr, forwardRelayer, reverseRelayer sdk.AccAddress, packetFee types.PacketFee) {
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

	// refund unused amount from the escrowed fee
	refundCoins := packetFee.Fee.Total().Sub(packetFee.Fee.RecvFee...).Sub(packetFee.Fee.AckFee...)
	k.distributeFee(ctx, refundAddr, refundAddr, refundCoins)
}

// DistributePacketFeesOnTimeout pays all the timeout fees for a given packetID while refunding the acknowledgement & receive fees to the refund account.
func (k Keeper) DistributePacketFeesOnTimeout(ctx context.Context, timeoutRelayer sdk.AccAddress, packetFees []types.PacketFee, packetID channeltypes.PacketId) {
	// use branched multistore for distribution of fees.
	// if the escrow account has insufficient balance then we want to avoid partially distributing fees
	if err := k.BranchService.Execute(ctx, func(ctx context.Context) error {
		for _, packetFee := range packetFees {
			if !k.EscrowAccountHasBalance(ctx, packetFee.Fee.Total()) {
				// NOTE: we lock the fee module on error return so that the state changes are persisted
				return ibcerrors.ErrInsufficientFunds
			}

			// check if refundAcc address works
			refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
			if err != nil {
				panic(fmt.Errorf("could not parse refundAcc %s to sdk.AccAddress", packetFee.RefundAddress))
			}

			k.distributePacketFeeOnTimeout(ctx, refundAddr, timeoutRelayer, packetFee)
		}

		return nil
	}); err != nil {
		if errors.Is(err, ibcerrors.ErrInsufficientFunds) {
			// if the escrow account does not have sufficient funds then there must exist a severe bug
			// the fee module should be locked until manual intervention fixes the issue
			// a locked fee module will simply skip fee logic, all channels will temporarily function as
			// fee disabled channels
			k.lockFeeModule(ctx)
			return
		}
	}

	// removing the fee from the store as the fee is now paid
	k.DeleteFeesInEscrow(ctx, packetID)
}

// distributePacketFeeOnTimeout pays the timeout fee to the timeout relayer and refunds the acknowledgement & receive fee.
func (k Keeper) distributePacketFeeOnTimeout(ctx context.Context, refundAddr, timeoutRelayer sdk.AccAddress, packetFee types.PacketFee) {
	// distribute fee for timeout relaying
	k.distributeFee(ctx, timeoutRelayer, refundAddr, packetFee.Fee.TimeoutFee)

	// refund unused amount from the escrowed fee
	refundCoins := packetFee.Fee.Total().Sub(packetFee.Fee.TimeoutFee...)
	k.distributeFee(ctx, refundAddr, refundAddr, refundCoins)
}

// distributeFee will attempt to distribute the escrowed fee to the receiver address.
// If the distribution fails for any reason (such as the receiving address being blocked),
// the state changes will be discarded.
func (k Keeper) distributeFee(ctx context.Context, receiver, refundAccAddress sdk.AccAddress, fee sdk.Coins) {
	// use branched multistore before trying to distribute fees
	if err := k.BranchService.Execute(ctx, func(ctx context.Context) error {
		err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, fee)
		if err != nil {
			if bytes.Equal(receiver, refundAccAddress) {
				// if sending to the refund address already failed, then return (no-op)
				return errorsmod.Wrapf(types.ErrRefundDistributionFailed, "receiver address: %s", receiver)
			}

			// if an error is returned from x/bank and the receiver is not the refundAccAddress
			// then attempt to refund the fee to the original sender
			err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, refundAccAddress, fee)
			if err != nil {
				// if sending to the refund address fails, no-op
				return errorsmod.Wrapf(types.ErrRefundDistributionFailed, "receiver address: %s", refundAccAddress)
			}

			if err := k.emitDistributeFeeEvent(ctx, refundAccAddress.String(), fee); err != nil {
				panic(err)
			}
		}

		return nil
	}); err != nil {
		k.Logger.Error("error distributing fee", "error", err.Error())
	}

	if err := k.emitDistributeFeeEvent(ctx, receiver.String(), fee); err != nil {
		panic(err)
	}
}

// RefundFeesOnChannelClosure will refund all fees associated with the given port and channel identifiers.
// If the escrow account runs out of balance then fee module will become locked as this implies the presence
// of a severe bug. When the fee module is locked, no fee distributions will be performed.
// Please see ADR 004 for more information.
func (k Keeper) RefundFeesOnChannelClosure(ctx context.Context, portID, channelID string) error {
	identifiedPacketFees := k.GetIdentifiedPacketFeesForChannel(ctx, portID, channelID)

	// use branched multistore for distribution of fees.
	// if the escrow account has insufficient balance then we want to avoid partially distributing fees
	if err := k.BranchService.Execute(ctx, func(ctx context.Context) error {
		for _, identifiedPacketFee := range identifiedPacketFees {
			var unRefundedFees []types.PacketFee
			for _, packetFee := range identifiedPacketFee.PacketFees {

				if !k.EscrowAccountHasBalance(ctx, packetFee.Fee.Total()) {
					// NOTE: we lock the fee module on error return so that the state changes are persisted
					return ibcerrors.ErrInsufficientFunds
				}

				refundAddr, err := sdk.AccAddressFromBech32(packetFee.RefundAddress)
				if err != nil {
					unRefundedFees = append(unRefundedFees, packetFee)
					continue
				}

				// refund all fees to refund address
				if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, refundAddr, packetFee.Fee.Total()); err != nil {
					unRefundedFees = append(unRefundedFees, packetFee)
					continue
				}
			}

			if len(unRefundedFees) > 0 {
				// update packet fees to keep only the unrefunded fees
				packetFees := types.NewPacketFees(unRefundedFees)
				k.SetFeesInEscrow(ctx, identifiedPacketFee.PacketId, packetFees)
			} else {
				k.DeleteFeesInEscrow(ctx, identifiedPacketFee.PacketId)
			}
		}

		return nil
	}); err != nil {
		if errors.Is(err, ibcerrors.ErrInsufficientFunds) {
			// if the escrow account does not have sufficient funds then there must exist a severe bug
			// the fee module should be locked until manual intervention fixes the issue
			// a locked fee module will simply skip fee logic, all channels will temporarily function as
			// fee disabled channels
			k.lockFeeModule(ctx)

			return nil // commit state changes to lock module and stop fee distribution
		}
	}

	return nil
}
