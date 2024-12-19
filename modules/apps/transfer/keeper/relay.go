package keeper

import (
	"context"
	"fmt"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/telemetry"
	internaltypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

// sendTransfer handles transfer sending logic. There are 2 possible cases:
//
// 1. Sender chain is acting as the source zone. The coins are transferred
// to an escrow address (i.e locked) on the sender chain and then transferred
// to the receiving chain through IBC TAO logic. It is expected that the
// receiving chain will mint vouchers to the receiving address.
//
// 2. Sender chain is acting as the sink zone. The coins (vouchers) are burned
// on the sender chain and then transferred to the receiving chain though IBC
// TAO logic. It is expected that the receiving chain, which had previously
// sent the original denomination, will unescrow the fungible token and send
// it to the receiving address.
//
// Another way of thinking of source and sink zones is through the token's
// timeline. Each send to any chain other than the one it was previously
// received from is a movement forwards in the token's timeline. This causes
// trace to be added to the token's history and the destination port and
// destination channel to be prefixed to the denomination. In these instances
// the sender chain is acting as the source zone. When the token is sent back
// to the chain it previously received from, the prefix is removed. This is
// a backwards movement in the token's timeline and the sender chain
// is acting as the sink zone.
//
// Example:
// These steps of transfer occur: A -> B -> C -> A -> C -> B -> A
//
// 1. A -> B : sender chain is source zone. Denom upon receiving: 'B/denom'
// 2. B -> C : sender chain is source zone. Denom upon receiving: 'C/B/denom'
// 3. C -> A : sender chain is source zone. Denom upon receiving: 'A/C/B/denom'
// 4. A -> C : sender chain is sink zone. Denom upon receiving: 'C/B/denom'
// 5. C -> B : sender chain is sink zone. Denom upon receiving: 'B/denom'
// 6. B -> A : sender chain is sink zone. Denom upon receiving: 'denom'
func (k Keeper) sendTransfer(
	ctx context.Context,
	sourcePort,
	sourceChannel string,
	coins sdk.Coins,
	sender sdk.AccAddress,
	receiver string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	memo string,
	hops []types.Hop,
) (uint64, error) {
	channel, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	appVersion, found := k.ics4Wrapper.GetAppVersion(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "application version not found for source port: %s and source channel: %s", sourcePort, sourceChannel)
	}

	if appVersion == types.V1 {
		// ics20-1 only supports a single coin, so if that is the current version, we must only process a single coin.
		if len(coins) > 1 {
			return 0, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot transfer multiple coins with %s", types.V1)
		}

		// ics20-1 does not support forwarding, so if that is the current version, we must reject the transfer.
		if len(hops) > 0 {
			return 0, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot forward coins with %s", types.V1)
		}
	}

	destinationPort := channel.Counterparty.PortId
	destinationChannel := channel.Counterparty.ChannelId

	// begin createOutgoingPacket logic
	// See spec for this logic: https://github.com/cosmos/ibc/tree/master/spec/app/ics-020-fungible-token-transfer#packet-relay

	tokens := make([]types.Token, 0, len(coins))

	for _, coin := range coins {
		// Using types.UnboundedSpendLimit allows us to send the entire balance of a given denom.
		if coin.Amount.Equal(types.UnboundedSpendLimit()) {
			coin.Amount = k.bankKeeper.SpendableCoin(ctx, sender, coin.Denom).Amount
			if coin.Amount.IsZero() {
				return 0, errorsmod.Wrapf(types.ErrInvalidAmount, "empty spendable balance for %s", coin.Denom)
			}
		}

		token, err := k.tokenFromCoin(ctx, coin)
		if err != nil {
			return 0, err
		}

		// NOTE: SendTransfer simply sends the denomination as it exists on its own
		// chain inside the packet data. The receiving chain will perform denom
		// prefixing as necessary.

		// if the denom is prefixed by the port and channel on which we are sending
		// the token, then we must be returning the token back to the chain they originated from
		if token.Denom.HasPrefix(sourcePort, sourceChannel) {
			// transfer the coins to the module account and burn them
			if err := k.bankKeeper.SendCoinsFromAccountToModule(
				ctx, sender, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				return 0, err
			}

			if err := k.bankKeeper.BurnCoins(
				ctx, k.authKeeper.GetModuleAddress(types.ModuleName), sdk.NewCoins(coin),
			); err != nil {
				// NOTE: should not happen as the module account was
				// retrieved on the step above and it has enough balance
				// to burn.
				panic(fmt.Errorf("cannot burn coins after a successful send to a module account: %v", err))
			}
		} else {
			// obtain the escrow address for the source channel end
			escrowAddress := types.GetEscrowAddress(sourcePort, sourceChannel)
			if err := k.escrowCoin(ctx, sender, escrowAddress, coin); err != nil {
				return 0, err
			}
		}

		tokens = append(tokens, token)
	}

	packetDataBytes, err := createPacketDataBytesFromVersion(appVersion, sender.String(), receiver, memo, tokens, hops)
	if err != nil {
		return 0, err
	}

	sequence, err := k.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, packetDataBytes)
	if err != nil {
		return 0, err
	}

	if err := k.EmitTransferEvent(ctx, sender.String(), receiver, tokens, memo, hops); err != nil {
		return 0, err
	}

	telemetry.ReportTransfer(sourcePort, sourceChannel, destinationPort, destinationChannel, tokens)

	return sequence, nil
}

// OnRecvPacket processes a cross chain fungible token transfer.
//
// If the sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// unescrowed and sent to the receiving address.
//
// In the case of packet forwarding, the packet is sent on the next hop as specified
// in the packet's ForwardingPacketData.
func (k Keeper) OnRecvPacket(ctx context.Context, packet channeltypes.Packet, data types.FungibleTokenPacketDataV2) error {
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

	if k.isBlockedAddr(receiver) {
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

			escrowAddress := types.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
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

			if err := k.EmitDenomEvent(ctx, token); err != nil {
				return err
			}

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
		if err := k.forwardPacket(ctx, data, packet, receivedCoins); err != nil {
			return err
		}
	}

	telemetry.ReportOnRecvPacket(packet, data.Tokens)

	// The ibc_module.go module will return the proper ack.
	return nil
}

// OnAcknowledgementPacket responds to the success or failure of a packet acknowledgment
// written on the receiving chain.
//
// If no forwarding occurs and the acknowledgement was a success then nothing occurs. Otherwise,
// if the acknowledgement failed, then the sender is refunded their tokens.
//
// If forwarding is used and the acknowledgement was a success, a successful acknowledgement is written
// for the forwarded packet. Otherwise, if the acknowledgement failed, after refunding the sender, the
// tokens of the forwarded packet that were received are in turn either refunded or burned.
func (k Keeper) OnAcknowledgementPacket(ctx context.Context, packet channeltypes.Packet, data types.FungibleTokenPacketDataV2, ack channeltypes.Acknowledgement) error {
	forwardedPacket, isForwarded := k.getForwardedPacket(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)

	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		if isForwarded {
			// Write a successful async ack for the forwardedPacket
			forwardAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
			return k.acknowledgeForwardedPacket(ctx, forwardedPacket, packet, forwardAck)
		}

		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	case *channeltypes.Acknowledgement_Error:
		// We refund the tokens from the escrow address to the sender
		if err := k.refundPacketTokens(ctx, packet, data); err != nil {
			return err
		}
		if isForwarded {
			// the forwarded packet has failed, thus the funds have been refunded to the intermediate address.
			// we must revert the changes that came from successfully receiving the tokens on our chain
			// before propagating the error acknowledgement back to original sender chain
			if err := k.revertForwardedPacket(ctx, forwardedPacket, data); err != nil {
				return err
			}

			forwardAck := internaltypes.NewForwardErrorAcknowledgement(packet, ack)
			return k.acknowledgeForwardedPacket(ctx, forwardedPacket, packet, forwardAck)
		}

		return nil
	default:
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected one of [%T, %T], got %T", channeltypes.Acknowledgement_Result{}, channeltypes.Acknowledgement_Error{}, ack.Response)
	}
}

// OnTimeoutPacket processes a transfer packet timeout.
//
// If no forwarding occurs, it refunds the tokens to the sender.
//
// If forwarding is used and the chain acted as a middle hop on a multihop transfer, after refunding
// the tokens to the sender, the tokens of the forwarded packet that were received are in turn
// either refunded or burned.
func (k Keeper) OnTimeoutPacket(ctx context.Context, packet channeltypes.Packet, data types.FungibleTokenPacketDataV2) error {
	if err := k.refundPacketTokens(ctx, packet, data); err != nil {
		return err
	}

	forwardedPacket, isForwarded := k.getForwardedPacket(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)
	if isForwarded {
		if err := k.revertForwardedPacket(ctx, forwardedPacket, data); err != nil {
			return err
		}

		forwardAck := internaltypes.NewForwardTimeoutAcknowledgement(packet)
		return k.acknowledgeForwardedPacket(ctx, forwardedPacket, packet, forwardAck)
	}

	return nil
}

// refundPacketTokens will unescrow and send back the tokens back to sender
// if the sending chain was the source chain. Otherwise, the sent tokens
// were burnt in the original send so new tokens are minted and sent to
// the sending address.
func (k Keeper) refundPacketTokens(ctx context.Context, packet channeltypes.Packet, data types.FungibleTokenPacketDataV2) error {
	// NOTE: packet data type already checked in handler.go

	sender, err := sdk.AccAddressFromBech32(data.Sender)
	if err != nil {
		return err
	}
	if k.isBlockedAddr(sender) {
		return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to receive funds", sender)
	}

	// escrow address for unescrowing tokens back to sender
	escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())

	moduleAccountAddr := k.authKeeper.GetModuleAddress(types.ModuleName)
	for _, token := range data.Tokens {
		coin, err := token.ToCoin()
		if err != nil {
			return err
		}

		// if the token we must refund is prefixed by the source port and channel
		// then the tokens were burnt when the packet was sent and we must mint new tokens
		if token.Denom.HasPrefix(packet.GetSourcePort(), packet.GetSourceChannel()) {
			// mint vouchers back to sender
			if err := k.bankKeeper.MintCoins(
				ctx, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				return err
			}

			if err := k.bankKeeper.SendCoins(ctx, moduleAccountAddr, sender, sdk.NewCoins(coin)); err != nil {
				panic(fmt.Errorf("unable to send coins from module to account despite previously minting coins to module account: %v", err))
			}
		} else {
			if err := k.unescrowCoin(ctx, escrowAddress, sender, coin); err != nil {
				return err
			}
		}
	}

	return nil
}

// escrowCoin will send the given coin from the provided sender to the escrow address. It will also
// update the total escrowed amount by adding the escrowed coin's amount to the current total escrow.
func (k Keeper) escrowCoin(ctx context.Context, sender, escrowAddress sdk.AccAddress, coin sdk.Coin) error {
	if err := k.bankKeeper.SendCoins(ctx, sender, escrowAddress, sdk.NewCoins(coin)); err != nil {
		// failure is expected for insufficient balances
		return err
	}

	// track the total amount in escrow keyed by denomination to allow for efficient iteration
	currentTotalEscrow := k.GetTotalEscrowForDenom(ctx, coin.GetDenom())
	newTotalEscrow := currentTotalEscrow.Add(coin)
	k.SetTotalEscrowForDenom(ctx, newTotalEscrow)

	return nil
}

// unescrowCoin will send the given coin from the escrow address to the provided receiver. It will also
// update the total escrow by deducting the unescrowed coin's amount from the current total escrow.
func (k Keeper) unescrowCoin(ctx context.Context, escrowAddress, receiver sdk.AccAddress, coin sdk.Coin) error {
	if err := k.bankKeeper.SendCoins(ctx, escrowAddress, receiver, sdk.NewCoins(coin)); err != nil {
		// NOTE: this error is only expected to occur given an unexpected bug or a malicious
		// counterparty module. The bug may occur in bank or any part of the code that allows
		// the escrow address to be drained. A malicious counterparty module could drain the
		// escrow address by allowing more tokens to be sent back then were escrowed.
		return errorsmod.Wrap(err, "unable to unescrow tokens, this may be caused by a malicious counterparty module or a bug: please open an issue on counterparty module")
	}

	// track the total amount in escrow keyed by denomination to allow for efficient iteration
	currentTotalEscrow := k.GetTotalEscrowForDenom(ctx, coin.GetDenom())
	newTotalEscrow := currentTotalEscrow.Sub(coin)
	k.SetTotalEscrowForDenom(ctx, newTotalEscrow)

	return nil
}

// tokenFromCoin constructs an IBC token given an SDK coin.
func (k Keeper) tokenFromCoin(ctx context.Context, coin sdk.Coin) (types.Token, error) {
	// if the coin does not have an IBC denom, return as is
	if !strings.HasPrefix(coin.Denom, "ibc/") {
		return types.Token{
			Denom:  types.NewDenom(coin.Denom),
			Amount: coin.Amount.String(),
		}, nil
	}

	// NOTE: denomination and hex hash correctness checked during msg.ValidateBasic
	hexHash := coin.Denom[len(types.DenomPrefix+"/"):]

	hash, err := types.ParseHexHash(hexHash)
	if err != nil {
		return types.Token{}, errorsmod.Wrap(types.ErrInvalidDenomForTransfer, err.Error())
	}

	denom, found := k.GetDenom(ctx, hash)
	if !found {
		return types.Token{}, errorsmod.Wrap(types.ErrDenomNotFound, hexHash)
	}

	return types.Token{
		Denom:  denom,
		Amount: coin.Amount.String(),
	}, nil
}

// createPacketDataBytesFromVersion creates the packet data bytes to be sent based on the application version.
func createPacketDataBytesFromVersion(appVersion, sender, receiver, memo string, tokens types.Tokens, hops []types.Hop) ([]byte, error) {
	switch appVersion {
	case types.V1:
		// Sanity check, tokens must always be of length 1 if using app version V1.
		if len(tokens) != 1 {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot transfer multiple coins with %s", types.V1)
		}

		token := tokens[0]
		packetData := types.NewFungibleTokenPacketData(token.Denom.Path(), token.Amount, sender, receiver, memo)

		if err := packetData.ValidateBasic(); err != nil {
			return nil, errorsmod.Wrapf(err, "failed to validate %s packet data", types.V1)
		}

		return packetData.GetBytes(), nil
	case types.V2:
		// If forwarding is needed, move memo to forwarding packet data and set packet.Memo to empty string.
		var forwardingPacketData types.ForwardingPacketData
		if len(hops) > 0 {
			forwardingPacketData = types.NewForwardingPacketData(memo, hops...)
			memo = ""
		}

		packetData := types.NewFungibleTokenPacketDataV2(tokens, sender, receiver, memo, forwardingPacketData)

		if err := packetData.ValidateBasic(); err != nil {
			return nil, errorsmod.Wrapf(err, "failed to validate %s packet data", types.V2)
		}

		return packetData.GetBytes(), nil
	default:
		return nil, errorsmod.Wrapf(types.ErrInvalidVersion, "app version must be one of %s", types.SupportedVersions)
	}
}
