package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/events"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/telemetry"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

func (k Keeper) OnRecvPacketV2(ctx sdk.Context, packet channeltypes.PacketV2, data types.FungibleTokenPacketDataV2) error {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return errorsmod.Wrapf(err, "error validating ICS-20 transfer packet data")
	}

	if !k.GetParams(ctx).ReceiveEnabled {
		return types.ErrReceiveDisabled
	}

	receiver, err := k.getReceiverFromPacketData(data)
	if err != nil {
		return err
	}

	if k.IsBlockedAddr(receiver) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to receive funds", receiver)
	}

	receivedCoins := make(sdk.Coins, 0, len(data.Tokens))
	for _, token := range data.Tokens {
		// parse the transfer amount
		transferAmount, ok := sdkmath.NewIntFromString(token.Amount)
		if !ok {
			return errorsmod.Wrapf(types.ErrInvalidAmount, "unable to parse transfer amount: %s", token.Amount)
		}

		// This is the prefix that would have been prefixed to the denomination
		// on sender chain IF and only if the token originally came from the
		// receiving chain.
		//
		// NOTE: We use SourcePort and SourceChannel here, because the counterparty
		// chain would have prefixed with DestPort and DestChannel when originally
		// receiving this token.
		if token.Denom.HasPrefix(packet.GetSourcePort(), packet.GetSourceChannel()) {
			// sender chain is not the source, unescrow tokens

			// remove prefix added by sender chain
			token.Denom.Trace = token.Denom.Trace[1:]

			coin := sdk.NewCoin(token.Denom.IBCDenom(), transferAmount)

			escrowAddress := types.GetEscrowAddress(packet.DestinationPort, packet.DestinationChannel)
			if err := k.unescrowCoin(ctx, escrowAddress, receiver, coin); err != nil {
				return err
			}

			// Appending token. The new denom has been computed
			receivedCoins = append(receivedCoins, coin)
		} else {
			// sender chain is the source, mint vouchers

			// since SendPacket did not prefix the denomination, we must add the destination port and channel to the trace
			trace := []types.Hop{types.NewHop(packet.DestinationPort, packet.DestinationChannel)}
			token.Denom.Trace = append(trace, token.Denom.Trace...)

			if !k.HasDenom(ctx, token.Denom.Hash()) {
				k.SetDenom(ctx, token.Denom)
			}

			voucherDenom := token.Denom.IBCDenom()
			if !k.bankKeeper.HasDenomMetaData(ctx, voucherDenom) {
				k.setDenomMetadata(ctx, token.Denom)
			}

			events.EmitDenomEvent(ctx, token)

			voucher := sdk.NewCoin(voucherDenom, transferAmount)

			// mint new tokens if the source of the transfer is the same chain
			if err := k.bankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(voucher),
			); err != nil {
				return errorsmod.Wrap(err, "failed to mint IBC tokens")
			}

			// send to receiver
			moduleAddr := k.authKeeper.GetModuleAddress(types.ModuleName)
			if err := k.bankKeeper.SendCoins(
				ctx, moduleAddr, receiver, sdk.NewCoins(voucher),
			); err != nil {
				return errorsmod.Wrapf(err, "failed to send coins to receiver %s", receiver.String())
			}

			receivedCoins = append(receivedCoins, voucher)
		}
	}

	if data.HasForwarding() {
		// we are now sending from the forward escrow address to the final receiver address.
		if err := k.forwardPacketV2(ctx, data, packet, receivedCoins); err != nil {
			return err
		}
	}

	telemetry.ReportOnRecvPacketV2(packet, data.Tokens)

	// The ibc_module.go module will return the proper ack.
	return nil
}
