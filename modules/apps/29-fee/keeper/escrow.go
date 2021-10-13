package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

// EscrowPacketFee sends the packet fee to the 29-fee module account to hold in escrow
func (k Keeper) EscrowPacketFee(ctx sdk.Context, refundAcc sdk.AccAddress, identifiedFee types.IdentifiedPacketFee) error {
	// check if the refund account exists
	hasRefundAcc := k.authKeeper.GetAccount(ctx, refundAcc)
	if hasRefundAcc == nil {
		return sdkerrors.Wrap(types.ErrRefundAccNotFound, fmt.Sprintf("Account with address: %s not found", refundAcc.String()))
	}

	fees := sdk.Coins{
		*identifiedFee.Fee.AckFee, *identifiedFee.Fee.ReceiveFee, *identifiedFee.Fee.TimeoutFee,
	}

	// check if refundAcc has balance for each fee
	for _, f := range fees {
		hasBalance := k.bankKeeper.HasBalance(ctx, refundAcc, f)
		if !hasBalance {
			return sdkerrors.Wrap(types.ErrBalanceNotFound, fmt.Sprintf("%s", refundAcc.String()))
		}
	}

	for _, coin := range fees {
		// escrow each fee with account module
		if err := k.bankKeeper.SendCoinsFromAccountToModule(
			ctx, refundAcc, types.ModuleName, sdk.Coins{coin},
		); err != nil {
			return err
		}
	}

	// Store fee in state for reference later
	// feeInEscrow/<channel-id>/packet/<sequence-id>/ -> Fee (timeout, ack, onrecv)
	k.SetFeeInEscrow(ctx, identifiedFee)
	return nil
}

// DistributeFee pays the acknowledgement fee & receive fee for a given packetId while refunding the timeout fee to the refund account associated with the Fee
func (k Keeper) DistributeFee(ctx sdk.Context, refundAcc, forwardRelayer, reverseRelayer sdk.AccAddress, packetID *channeltypes.PacketId) error {
	// check if there is a Fee in escrow for the given packetId
	feeInEscrow, found := k.GetFeeInEscrow(ctx, packetID)
	if !found {
		return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
	}

	// get module accAddr
	feeModuleAccAddr := k.authKeeper.GetModuleAddress(types.ModuleName)

	// send ack fee to reverse relayer
	err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, reverseRelayer, sdk.Coins{*feeInEscrow.Fee.AckFee})
	if err != nil {
		return sdkerrors.Wrap(types.ErrPayingFee, fmt.Sprintf("Error sending coin with Denom: %s for Amount: %d", feeInEscrow.Fee.AckFee.Denom, feeInEscrow.Fee.AckFee.Amount))
	}

	// send receive fee to forward relayer
	err = k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, forwardRelayer, sdk.Coins{*feeInEscrow.Fee.ReceiveFee})
	if err != nil {
		return sdkerrors.Wrap(types.ErrPayingFee, fmt.Sprintf("Error sending coin with Denom: %s for Amount: %d", feeInEscrow.Fee.ReceiveFee.Denom, feeInEscrow.Fee.ReceiveFee.Amount))
	}

	// refund timeout fee to refundAddr
	err = k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, sdk.Coins{*feeInEscrow.Fee.TimeoutFee})
	if err != nil {
		return sdkerrors.Wrap(types.ErrRefundingFee, fmt.Sprintf("Error sending coin with Denom: %s for Amount: %d", feeInEscrow.Fee.TimeoutFee.Denom, feeInEscrow.Fee.TimeoutFee.Amount))
	}

	// set fee as an empty struct (if we reach this point Fee is paid in full)
	identifiedPacket := types.IdentifiedPacketFee{PacketId: packetID, Fee: &types.Fee{}, Relayers: []string{}}
	k.SetFeeInEscrow(ctx, identifiedPacket)

	return nil
}

// DistributeFeeTimeout pays the timeout fee for a given packetId while refunding the acknowledgement fee & receive fee to the refund account associated with the Fee
func (k Keeper) DistributeFeeTimeout(ctx sdk.Context, refundAcc, reverseRelayer sdk.AccAddress, packetID *channeltypes.PacketId) error {
	// check if there is a Fee in escrow for the given packetId
	feeInEscrow, hasFee := k.GetFeeInEscrow(ctx, packetID)
	if !hasFee {
		return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
	}

	// get module accAddr
	feeModuleAccAddr := k.authKeeper.GetModuleAddress(types.ModuleName)

	// refund the ack fee
	err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, sdk.Coins{*feeInEscrow.Fee.AckFee})
	if err != nil {
		return sdkerrors.Wrap(types.ErrRefundingFee, fmt.Sprintf("Error sending coin with Denom: %s for Amount: %d", feeInEscrow.Fee.AckFee.Denom, feeInEscrow.Fee.AckFee.Amount))
	}

	// refund the receive fee
	err = k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, sdk.Coins{*feeInEscrow.Fee.ReceiveFee})
	if err != nil {
		return sdkerrors.Wrap(types.ErrRefundingFee, fmt.Sprintf("Error sending coin with Denom: %s for Amount: %d", feeInEscrow.Fee.ReceiveFee.Denom, feeInEscrow.Fee.ReceiveFee.Amount))
	}

	// pay the timeout fee to the reverse relayer
	err = k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, reverseRelayer, sdk.Coins{*feeInEscrow.Fee.TimeoutFee})
	if err != nil {
		return sdkerrors.Wrap(types.ErrPayingFee, fmt.Sprintf("Error sending coin with Denom: %s for Amount: %d", feeInEscrow.Fee.TimeoutFee.Denom, feeInEscrow.Fee.TimeoutFee.Amount))
	}

	// set fee as an empty struct (if we reach this point Fee is paid in full)
	identifiedPacket := types.IdentifiedPacketFee{PacketId: packetID, Fee: &types.Fee{}, Relayers: []string{}}
	k.SetFeeInEscrow(ctx, identifiedPacket)

	return nil
}
