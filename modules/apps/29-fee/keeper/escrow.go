package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

// TODO: add optional relayers arr
func (k Keeper) EscrowPacketFee(ctx sdk.Context, refundAcc sdk.AccAddress, fee types.Fee, packetID channeltypes.PacketId) error {
	// check if the refund account exists
	hasRefundAcc := k.authKeeper.GetAccount(ctx, refundAcc)
	if hasRefundAcc == nil {
		return sdkerrors.Wrap(types.ErrRefundAccNotFound, fmt.Sprintf("Account with address: %s not found", refundAcc.String()))
	}

	fees := sdk.Coins{
		*fee.AckFee, *fee.ReceiveFee, *fee.TimeoutFee,
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
	// feeInEscrow/<refund-account>/<channel-id>/packet/<sequence-id>/ -> Fee (timeout, ack, onrecv)
	k.SetFeeInEscrow(ctx, fee, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	return nil
}

func (k Keeper) payFee(ctx sdk.Context, refundAcc, forwardRelayer, reverseRelayer sdk.AccAddress, packetID channeltypes.PacketId) error {
	// check if there is a Fee in escrow for the given packetId
	feeInEscrow, hasFee := k.GetFeeInEscrow(ctx, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	if !hasFee {
		return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
	}

	// get module accAddr
	feeModuleAccAddr := k.authKeeper.GetModuleAddress(types.ModuleName)

	// send ack fee to forward relayer
	if feeInEscrow.AckFee != nil {
		err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, forwardRelayer, sdk.Coins{*feeInEscrow.AckFee})
		if err != nil {
			return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
		}
		feeInEscrow.AckFee = nil
		k.SetFeeInEscrow(ctx, feeInEscrow, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	}

	// send ack fee to forward relayer
	if feeInEscrow.ReceiveFee != nil {
		err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, reverseRelayer, sdk.Coins{*feeInEscrow.AckFee})
		if err != nil {
			return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
		}
		feeInEscrow.ReceiveFee = nil
		k.SetFeeInEscrow(ctx, feeInEscrow, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	}

	// refund timeout fee to refundAddr
	if feeInEscrow.TimeoutFee != nil {
		err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, sdk.Coins{*feeInEscrow.AckFee})
		if err != nil {
			return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
		}
		// set fee as an empty struct (if we reach this point Fee is paid in full)
		k.SetFeeInEscrow(ctx, types.Fee{}, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	}

	return nil
}

func (k Keeper) payFeeTimeout(ctx sdk.Context, refundAcc, forwardRelayer, reverseRelayer sdk.AccAddress, packetID channeltypes.PacketId) error {
	// check if there is a Fee in escrow for the given packetId
	feeInEscrow, hasFee := k.GetFeeInEscrow(ctx, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	if !hasFee {
		return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
	}

	// get module accAddr
	feeModuleAccAddr := k.authKeeper.GetModuleAddress(types.ModuleName)

	// send ack fee to forward relayer
	if feeInEscrow.AckFee != nil {
		err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, sdk.Coins{*feeInEscrow.AckFee})
		if err != nil {
			return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
		}
		feeInEscrow.AckFee = nil
		k.SetFeeInEscrow(ctx, feeInEscrow, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	}

	// send ack fee to forward relayer
	if feeInEscrow.ReceiveFee != nil {
		err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, refundAcc, sdk.Coins{*feeInEscrow.AckFee})
		if err != nil {
			return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
		}
		feeInEscrow.ReceiveFee = nil
		k.SetFeeInEscrow(ctx, feeInEscrow, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	}

	// refund timeout fee to refundAddr
	if feeInEscrow.TimeoutFee != nil {
		err := k.bankKeeper.SendCoins(ctx, feeModuleAccAddr, forwardRelayer, sdk.Coins{*feeInEscrow.AckFee})
		if err != nil {
			return sdkerrors.Wrap(types.ErrFeeNotFound, fmt.Sprintf("%s", refundAcc.String()))
		}
		// set fee as an empty struct (if we reach this point Fee is paid in full)
		k.SetFeeInEscrow(ctx, types.Fee{}, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	}

	return nil
}
