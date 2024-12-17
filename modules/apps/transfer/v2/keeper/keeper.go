package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transferkeeper "github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	channelkeeperv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/keeper"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

type Keeper struct {
	transferkeeper.Keeper
	channelKeeperV2 *channelkeeperv2.Keeper
}

func NewKeeper(transferKeeper transferkeeper.Keeper, channelKeeperV2 *channelkeeperv2.Keeper) *Keeper {
	return &Keeper{
		Keeper:          transferKeeper,
		channelKeeperV2: channelKeeperV2,
	}
}

func (k *Keeper) OnAcknowledgementPacket(ctx context.Context, sourcePort, sourceChannel string, data types.FungibleTokenPacketDataV2, ack channeltypes.Acknowledgement) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	case *channeltypes.Acknowledgement_Error:
		// We refund the tokens from the escrow address to the sender
		return k.refundPacketTokens(ctx, sourcePort, sourceChannel, data)
	default:
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected one of [%T, %T], got %T", channeltypes.Acknowledgement_Result{}, channeltypes.Acknowledgement_Error{}, ack.Response)
	}
}

func (k *Keeper) OnTimeoutPacket(ctx context.Context, sourcePort, sourceChannel string, data types.FungibleTokenPacketDataV2) error {
	return k.refundPacketTokens(ctx, sourcePort, sourceChannel, data)
}

func (k Keeper) refundPacketTokens(ctx context.Context, sourcePort, sourceChannel string, data types.FungibleTokenPacketDataV2) error {
	// NOTE: packet data type already checked in handler.go

	sender, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return err
	}
	if k.IsBlockedAddr(sender) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to receive funds", sender)
	}

	// escrow address for unescrowing tokens back to sender
	escrowAddress := types.GetEscrowAddress(sourcePort, sourceChannel)

	moduleAccountAddr := k.AuthKeeper.GetModuleAddress(types.ModuleName)
	for _, token := range data.Tokens {
		coin, err := token.ToCoin()
		if err != nil {
			return err
		}

		// if the token we must refund is prefixed by the source port and channel
		// then the tokens were burnt when the packet was sent and we must mint new tokens
		if token.Denom.HasPrefix(sourcePort, sourceChannel) {
			// mint vouchers back to sender
			if err := k.BankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				return err
			}

			if err := k.BankKeeper.SendCoins(ctx, moduleAccountAddr, sender, sdk.NewCoins(coin)); err != nil {
				panic(fmt.Errorf("unable to send coins from module to account despite previously minting coins to module account: %v", err))
			}
		} else {
			if err := k.UnescrowCoin(ctx, escrowAddress, sender, coin); err != nil {
				return err
			}
		}
	}

	return nil
}
