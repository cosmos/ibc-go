package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/events"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	packetservertypes "github.com/cosmos/ibc-go/v9/modules/core/packet-server/types"
)

func (k Keeper) OnRecvPacketV2(ctx context.Context, packet channeltypes.PacketV2, payload channeltypes.Payload, data types.FungibleTokenPacketDataV2) error {
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
		// TODO figure out what to use for source port in denom trace
		if token.Denom.HasPrefix(types.ModuleName, packet.GetSourceId()) {
			// sender chain is not the source, unescrow tokens

			// remove prefix added by sender chain
			token.Denom.Trace = token.Denom.Trace[1:]

			coin := sdk.NewCoin(token.Denom.IBCDenom(), transferAmount)

			escrowAddress := types.GetEscrowAddress(types.ModuleName, packet.DestinationId)
			if err := k.unescrowCoin(ctx, escrowAddress, receiver, coin); err != nil {
				return err
			}

			// Appending token. The new denom has been computed
			receivedCoins = append(receivedCoins, coin)
		} else {
			// sender chain is the source, mint vouchers

			// since SendPacket did not prefix the denomination, we must add the destination port and channel to the trace
			trace := []types.Hop{types.NewHop(types.ModuleName, packet.DestinationId)}
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

	// if data.HasForwarding() {
	// 	// we are now sending from the forward escrow address to the final receiver address.
	// 	if err := k.forwardPacketV2(ctx, data, packet, receivedCoins); err != nil {
	// 		return err
	// 	}
	// }

	// telemetry.ReportOnRecvPacketV2(packet, data.Tokens)

	// The ibc_module.go module will return the proper ack.
	return nil
}

func (k Keeper) OnAcknowledgementPacketV2(ctx context.Context, packet channeltypes.PacketV2, data types.FungibleTokenPacketDataV2, ack channeltypes.Acknowledgement) error {
	// forwardedPacket, isForwarded := k.getForwardedPacketV2(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)

	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		// if isForwarded {
		// 	// Write a successful async ack for the forwardedPacket
		// 	forwardAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
		// 	return k.acknowledgeForwardedPacketV2(ctx, forwardedPacket, packet, forwardAck)
		// }

		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	case *channeltypes.Acknowledgement_Error:
		// We refund the tokens from the escrow address to the sender
		if err := k.refundPacketTokensV2(ctx, packet, data); err != nil {
			return err
		}
		// if isForwarded {
		// 	// the forwarded packet has failed, thus the funds have been refunded to the intermediate address.
		// 	// we must revert the changes that came from successfully receiving the tokens on our chain
		// 	// before propagating the error acknowledgement back to original sender chain
		// 	if err := k.revertForwardedPacketV2(ctx, forwardedPacket, data); err != nil {
		// 		return err
		// 	}

		// 	forwardAck := internaltypes.NewForwardErrorAcknowledgementV2(packet, ack)
		// 	return k.acknowledgeForwardedPacketV2(ctx, forwardedPacket, packet, forwardAck)
		// }

		return nil
	default:
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected one of [%T, %T], got %T", channeltypes.Acknowledgement_Result{}, channeltypes.Acknowledgement_Error{}, ack.Response)
	}
}

func (k Keeper) OnTimeoutPacketV2(ctx context.Context, packet channeltypes.PacketV2, data types.FungibleTokenPacketDataV2) error {
	if err := k.refundPacketTokensV2(ctx, packet, data); err != nil {
		return err
	}

	// forwardedPacket, isForwarded := k.getForwardedPacketV2(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)
	// if isForwarded {
	// 	if err := k.revertForwardedPacketV2(ctx, forwardedPacket, data); err != nil {
	// 		return err
	// 	}

	// 	forwardAck := internaltypes.NewForwardTimeoutAcknowledgementV2(packet)
	// 	return k.acknowledgeForwardedPacketV2(ctx, forwardedPacket, packet, forwardAck)
	// }

	return nil
}

func (k Keeper) OnSendPacket(
	ctx context.Context,
	sourceID string,
	packetData types.FungibleTokenPacketDataV2,
	sender sdk.AccAddress,
) error {
	_, ok := k.packetServerKeeper.GetCounterparty(ctx, sourceID)
	if !ok {
		return errorsmod.Wrap(packetservertypes.ErrCounterpartyNotFound, sourceID)
	}

	var coins sdk.Coins
	for _, token := range packetData.Tokens {
		transferAmount, ok := sdkmath.NewIntFromString(token.Amount)
		if !ok {
			return errorsmod.Wrapf(types.ErrInvalidAmount, "unable to parse transfer amount: %s", token.Amount)
		}

		coins = append(coins, sdk.NewCoin(token.Denom.IBCDenom(), transferAmount))
	}

	if err := k.bankKeeper.IsSendEnabledCoins(ctx, coins...); err != nil {
		return errorsmod.Wrapf(types.ErrSendDisabled, err.Error())
	}

	// begin createOutgoingPacket logic
	// See spec for this logic: https://github.com/cosmos/ibc/tree/master/spec/app/ics-020-fungible-token-transfer#packet-relay
	tokens := make([]types.Token, 0, len(coins))

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	for _, coin := range coins {
		// Using types.UnboundedSpendLimit allows us to send the entire balance of a given denom.
		if coin.Amount.Equal(types.UnboundedSpendLimit()) {
			coin.Amount = k.bankKeeper.GetBalance(ctx, sender, coin.Denom).Amount
		}

		token, err := k.tokenFromCoin(sdkCtx, coin)
		if err != nil {
			return err
		}

		// NOTE: SendTransfer simply sends the denomination as it exists on its own
		// chain inside the packet data. The receiving chain will perform denom
		// prefixing as necessary.

		// if the denom is prefixed by the port and channel on which we are sending
		// the token, then we must be returning the token back to the chain they originated from
		if token.Denom.HasPrefix(types.ModuleName, sourceID) {
			// transfer the coins to the module account and burn them
			if err := k.bankKeeper.SendCoinsFromAccountToModule(
				ctx, sender, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				return err
			}

			if err := k.bankKeeper.BurnCoins(
				ctx, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				// NOTE: should not happen as the module account was
				// retrieved on the step above and it has enough balance
				// to burn.
				panic(fmt.Errorf("cannot burn coins after a successful send to a module account: %v", err))
			}
		} else {
			// obtain the escrow address for the source channel end
			escrowAddress := types.GetEscrowAddress(types.ModuleName, sourceID)
			if err := k.escrowCoin(ctx, sender, escrowAddress, coin); err != nil {
				return err
			}
		}

		tokens = append(tokens, token)
	}

	// events.EmitTransferEvent(ctx, sender.String(), packetData.Receiver, tokens, packetData.Memo, packetData.Forwarding.Hops)

	// telemetry.ReportTransfer(sourcePort, sourceChannel, destinationPort, destinationChannel, tokens)

	return nil
}
