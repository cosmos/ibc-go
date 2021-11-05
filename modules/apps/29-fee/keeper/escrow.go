package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

// EscrowPacketFee sends the packet fee to the 29-fee module account to hold in escrow
func (k Keeper) EscrowPacketFee(ctx sdk.Context, refundAcc sdk.AccAddress, identifiedFee *types.IdentifiedPacketFee) error {
	// check if the refund account exists
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
func (k Keeper) DistributeFee(ctx sdk.Context, refundAcc, forwardRelayer, reverseRelayer sdk.AccAddress, packetID *channeltypes.PacketId) error {
	// check if there is a Fee in escrow for the given packetId
	feeInEscrow, found := k.GetFeeInEscrow(ctx, packetID)
	if !found {
		return sdkerrors.Wrapf(types.ErrFeeNotFound, "with channelID %s, sequence %d", packetID.ChannelId, packetID.Sequence)
	}

	// get module accAddr
	feeModuleAccAddr := k.authKeeper.GetModuleAddress(types.ModuleName)

	// send receive fee to forward relayer
	err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, forwardRelayer, feeInEscrow.Fee.ReceiveFee)
	if err != nil {
		return sdkerrors.Wrap(err, "failed to send fee to forward relayer")
	}

	// send ack fee to reverse relayer
	err = k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, reverseRelayer, feeInEscrow.Fee.AckFee)
	if err != nil {
		return sdkerrors.Wrap(err, "error sending fee to reverse relayer")
	}

	// refund timeout fee to refundAddr
	err = k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, feeInEscrow.Fee.TimeoutFee)
	if err != nil {
		return sdkerrors.Wrap(err, "error refunding timeout fee")
	}

	// removes the fee from the store as fee is now paid
	k.DeleteFeeInEscrow(ctx, packetID)

	return nil
}

// DistributeFeeTimeout pays the timeout fee for a given packetId while refunding the acknowledgement fee & receive fee to the refund account associated with the Fee
func (k Keeper) DistributeFeeTimeout(ctx sdk.Context, refundAcc, timeoutRelayer sdk.AccAddress, packetID *channeltypes.PacketId) error {
	// check if there is a Fee in escrow for the given packetId
	feeInEscrow, found := k.GetFeeInEscrow(ctx, packetID)
	if !found {
		return sdkerrors.Wrap(types.ErrFeeNotFound, refundAcc.String())
	}

	// get module accAddr
	feeModuleAccAddr := k.authKeeper.GetModuleAddress(types.ModuleName)

	// refund the receive fee
	err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, feeInEscrow.Fee.ReceiveFee)
	if err != nil {
		return sdkerrors.Wrap(err, "error refunding receive fee")
	}

	// refund the ack fee
	err = k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, feeInEscrow.Fee.AckFee)
	if err != nil {
		return sdkerrors.Wrap(err, "error refunding ack fee")
	}

	// pay the timeout fee to the reverse relayer
	err = k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, timeoutRelayer, feeInEscrow.Fee.TimeoutFee)
	if err != nil {
		return sdkerrors.Wrap(err, "error sending fee to timeout relayer")
	}

	// removes the fee from the store as fee is now paid
	k.DeleteFeeInEscrow(ctx, packetID)

	return nil
}
