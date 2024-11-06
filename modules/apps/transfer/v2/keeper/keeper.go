package keeper

import (
	"context"
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/events"
	transferkeeper "github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channelkeeperv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/keeper"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

type Keeper struct {
	transferkeeper.Keeper
	channelKeeperV2  *channelkeeperv2.Keeper
	msgServiceRouter *baseapp.MsgServiceRouter
}

func NewKeeper(transferKeeper transferkeeper.Keeper, channelKeeperV2 *channelkeeperv2.Keeper, msgServiceRouter *baseapp.MsgServiceRouter) *Keeper {
	return &Keeper{
		Keeper:           transferKeeper,
		channelKeeperV2:  channelKeeperV2,
		msgServiceRouter: msgServiceRouter,
	}
}

func (k *Keeper) OnSendPacket(ctx context.Context, sourceChannel string, payload channeltypesv2.Payload, data types.FungibleTokenPacketDataV2, sender sdk.AccAddress) error {
	// TODO unwind logic for forwarding

	for _, token := range data.Tokens {
		coin, err := token.ToCoin()
		if err != nil {
			return err
		}

		if coin.Amount.Equal(types.UnboundedSpendLimit()) {
			coin.Amount = k.BankKeeper.GetBalance(ctx, sender, coin.Denom).Amount
		}

		// NOTE: SendTransfer simply sends the denomination as it exists on its own
		// chain inside the packet data. The receiving chain will perform denom
		// prefixing as necessary.

		// if the denom is prefixed by the port and channel on which we are sending
		// the token, then we must be returning the token back to the chain they originated from
		if token.Denom.HasPrefix(payload.SourcePort, sourceChannel) {
			// transfer the coins to the module account and burn them
			if err := k.BankKeeper.SendCoinsFromAccountToModule(
				ctx, sender, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				return err
			}

			if err := k.BankKeeper.BurnCoins(
				ctx, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				// NOTE: should not happen as the module account was
				// retrieved on the step above and it has enough balance
				// to burn.
				panic(fmt.Errorf("cannot burn coins after a successful send to a module account: %v", err))
			}
		} else {
			// obtain the escrow address for the source channel end
			escrowAddress := types.GetEscrowAddress(payload.SourcePort, sourceChannel)
			if err := k.EscrowCoin(ctx, sender, escrowAddress, coin); err != nil {
				return err
			}
		}
	}

	// TODO: events
	// events.EmitTransferEvent(ctx, sender.String(), receiver, tokens, memo, hops)

	// TODO: telemetry
	// telemetry.ReportTransfer(sourcePort, sourceChannel, destinationPort, destinationChannel, tokens)

	return nil
}

func (k *Keeper) OnRecvPacket(ctx context.Context, sourceChannel, destChannel string, payload channeltypesv2.Payload, data types.FungibleTokenPacketDataV2) error {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return errorsmod.Wrapf(err, "error validating ICS-20 transfer packet data")
	}

	if !k.GetParams(ctx).ReceiveEnabled {
		return types.ErrReceiveDisabled
	}

	receiver, err := sdk.AccAddressFromBech32(data.Receiver)
	if err != nil {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "failed to decode receiver address %s: %v", data.Receiver, err)
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
		if token.Denom.HasPrefix(payload.SourcePort, sourceChannel) {
			// sender chain is not the source, unescrow tokens

			// remove prefix added by sender chain
			token.Denom.Trace = token.Denom.Trace[1:]

			coin := sdk.NewCoin(token.Denom.IBCDenom(), transferAmount)

			escrowAddress := types.GetEscrowAddress(payload.DestinationPort, destChannel)
			if err := k.UnescrowCoin(ctx, escrowAddress, receiver, coin); err != nil {
				return err
			}

			// Appending token. The new denom has been computed
			receivedCoins = append(receivedCoins, coin)
		} else {
			// sender chain is the source, mint vouchers

			// since SendPacket did not prefix the denomination, we must add the destination port and channel to the trace
			trace := []types.Hop{types.NewHop(payload.DestinationPort, destChannel)}
			token.Denom.Trace = append(trace, token.Denom.Trace...)

			if !k.HasDenom(ctx, token.Denom.Hash()) {
				k.SetDenom(ctx, token.Denom)
			}

			voucherDenom := token.Denom.IBCDenom()
			if !k.BankKeeper.HasDenomMetaData(ctx, voucherDenom) {
				k.SetDenomMetadata(ctx, token.Denom)
			}

			events.EmitDenomEvent(ctx, token)

			voucher := sdk.NewCoin(voucherDenom, transferAmount)

			// mint new tokens if the source of the transfer is the same chain
			if err := k.BankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(voucher),
			); err != nil {
				return errorsmod.Wrap(err, "failed to mint IBC tokens")
			}

			// send to receiver
			moduleAddr := k.AuthKeeper.GetModuleAddress(types.ModuleName)
			if err := k.BankKeeper.SendCoins(
				ctx, moduleAddr, receiver, sdk.NewCoins(voucher),
			); err != nil {
				return errorsmod.Wrapf(err, "failed to send coins to receiver %s", receiver.String())
			}

			receivedCoins = append(receivedCoins, voucher)
		}
	}

	// TODO: forwarding
	//if data.HasForwarding() {
	//	// we are now sending from the forward escrow address to the final receiver address.
	//	if err := k.forwardPacket(ctx, data, packet, receivedCoins); err != nil {
	//		return err
	//	}
	//}

	// TODO: telemetry
	//telemetry.ReportOnRecvPacket(packet, data.Tokens)

	// The ibc_module.go module will return the proper ack.
	return nil
}

// TODO move to a different file
// forwardPacket forwards a fungible FungibleTokenPacketDataV2 to the next hop in the forwarding path.
func (k Keeper) forwardPacket(ctx context.Context, data types.FungibleTokenPacketDataV2, payload channeltypesv2.Payload, timeoutTimestamp uint64, receivedCoins sdk.Coins) error {
	newForwardingPacketData := types.NewForwardingPacketData(data.Forwarding.DestinationMemo)
	if len(data.Forwarding.Hops) > 1 {
		// remove the first hop since we are going to send to the first hop now and we want to propagate the rest of the hops to the receiver
		newForwardingPacketData.Hops = data.Forwarding.Hops[1:]
	}

	memo := data.Memo
	if len(newForwardingPacketData.Hops) > 0 {
		memo = ""
	}

	// sending from module account (used as a temporary forward escrow) to the original receiver address.
	sender := k.AuthKeeper.GetModuleAddress(types.ModuleName)

	newPacketData := types.NewFungibleTokenPacketDataV2(data.Tokens, data.Sender, data.Receiver, memo, newForwardingPacketData)

	pdBz, err := newPacketData.Marshal()
	if err != nil {
		return err
	}
	payload.Value = pdBz

	msg := channeltypesv2.NewMsgSendPacket(
		data.Forwarding.Hops[0].ChannelId,
		timeoutTimestamp,
		sender.String(),
		payload,
	)

	handler := k.msgServiceRouter.Handler(&channeltypesv2.MsgSendPacket{})
	resp, err := handler(sdk.UnwrapSDKContext(ctx), msg)
	if err != nil {
		return err
	}
	_ = resp
	// k.setForwardedPacketSequence(ctx, data.Forwarding.Hops[0].PortId, data.Forwarding.Hops[0].ChannelId, resp.Sequence)
	return nil
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
