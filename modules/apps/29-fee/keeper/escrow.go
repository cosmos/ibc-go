package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

func (k Keeper) escrowPacketFee(ctx sdk.Context, refundAcc sdk.AccAddress, fee types.Fee, packetID channeltypes.PacketId) error {
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
			return sdkerrors.Wrap(types.ErrBalanceNotFound, fmt.Sprintf("", refundAcc.String()))
		}
	}

	// escrow each fee with account module
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, refundAcc, types.ModuleName, fees,
	); err != nil {
		return err
	}

	// Store fee in state for reference later
	// <refund-account>/<channel-id>/packet/<sequence-id>/ -> Fee (timeout, ack, onrecv)
	k.SetFeeInEscrow(ctx, fee, refundAcc.String(), packetID.ChannelId, packetID.Sequence)
	return nil
}

//TODO: implement
func (k Keeper) PayFee(ctx sdk.Context, refundAcc sdk.AccAddress, fee types.Fee, packetID channeltypes.PacketId) error {
	return nil
}
