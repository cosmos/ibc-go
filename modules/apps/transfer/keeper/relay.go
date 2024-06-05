package keeper

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-metrics"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/internal/events"
	internaltelemetry "github.com/cosmos/ibc-go/v8/modules/apps/transfer/internal/telemetry"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	coretypes "github.com/cosmos/ibc-go/v8/modules/core/types"
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
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	coins sdk.Coins,
	sender sdk.AccAddress,
	receiver string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	memo string,
) (uint64, error) {
	channel, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	appVersion, found := k.ics4Wrapper.GetAppVersion(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "application version not found for source port: %s and source channel: %s", sourcePort, sourceChannel)
	}

	// ics20-1 only supports a single coin, so if that is the current version, we must only process a single coin.
	if appVersion == types.V1 && len(coins) > 1 {
		return 0, errorsmod.Wrapf(ibcerrors.ErrInvalidRequest, "cannot transfer multiple coins with ics20-1")
	}

	destinationPort := channel.Counterparty.PortId
	destinationChannel := channel.Counterparty.ChannelId

	labels := []metrics.Label{
		telemetry.NewLabel(coretypes.LabelDestinationPort, destinationPort),
		telemetry.NewLabel(coretypes.LabelDestinationChannel, destinationChannel),
	}

	// begin createOutgoingPacket logic
	// See spec for this logic: https://github.com/cosmos/ibc/tree/master/spec/app/ics-020-fungible-token-transfer#packet-relay
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return 0, errorsmod.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	var tokens []types.Token

	for _, coin := range coins {
		token, err := k.tokenFromCoin(ctx, coin)
		if err != nil {
			return 0, err
		}

		// NOTE: SendTransfer simply sends the denomination as it exists on its own
		// chain inside the packet data. The receiving chain will perform denom
		// prefixing as necessary.

		if token.Denom.SenderChainIsSource(sourcePort, sourceChannel) {
			labels = append(labels, telemetry.NewLabel(coretypes.LabelSource, "true"))

			// obtain the escrow address for the source channel end
			escrowAddress := types.GetEscrowAddress(sourcePort, sourceChannel)
			if err := k.escrowCoin(ctx, sender, escrowAddress, coin); err != nil {
				return 0, err
			}

		} else {
			labels = append(labels, telemetry.NewLabel(coretypes.LabelSource, "false"))

			// transfer the coins to the module account and burn them
			if err := k.bankKeeper.SendCoinsFromAccountToModule(
				ctx, sender, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				return 0, err
			}

			if err := k.bankKeeper.BurnCoins(
				ctx, types.ModuleName, sdk.NewCoins(coin),
			); err != nil {
				// NOTE: should not happen as the module account was
				// retrieved on the step above and it has enough balance
				// to burn.
				panic(fmt.Errorf("cannot burn coins after a successful send to a module account: %v", err))
			}
		}

		tokens = append(tokens, token)
	}

	packetDataBytes := createPacketDataBytesFromVersion(appVersion, sender.String(), receiver, memo, tokens)

	sequence, err := k.ics4Wrapper.SendPacket(ctx, channelCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, packetDataBytes)
	if err != nil {
		return 0, err
	}

	events.EmitTransferEvent(ctx, sender.String(), receiver, tokens, memo)

	defer internaltelemetry.ReportTransferTelemetry(tokens, labels)

	return sequence, nil
}

// OnRecvPacket processes a cross chain fungible token transfer. If the
// sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// unescrowed and sent to the receiving address.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketDataV2) error {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return errorsmod.Wrapf(err, "error validating ICS-20 transfer packet data")
	}

	if !k.GetParams(ctx).ReceiveEnabled {
		return types.ErrReceiveDisabled
	}

	// decode the receiver address
	receiver, err := sdk.AccAddressFromBech32(data.Receiver)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to decode receiver address: %s", data.Receiver)
	}

	for _, token := range data.Tokens {
		labels := []metrics.Label{
			telemetry.NewLabel(coretypes.LabelSourcePort, packet.GetSourcePort()),
			telemetry.NewLabel(coretypes.LabelSourceChannel, packet.GetSourceChannel()),
		}

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
		// receiving this coin as seen in the "sender chain is the source" condition.
		if token.Denom.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel()) {
			// sender chain is not the source, unescrow tokens

			// remove prefix added by sender chain
			token.Denom.Trace = token.Denom.Trace[1:]

			coin := sdk.NewCoin(token.Denom.IBCDenom(), transferAmount)

			if k.bankKeeper.BlockedAddr(receiver) {
				return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to receive funds", receiver)
			}

			escrowAddress := types.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
			if err := k.unescrowCoin(ctx, escrowAddress, receiver, coin); err != nil {
				return err
			}

			denomPath := token.Denom.Path()
			labels = append(labels, telemetry.NewLabel(coretypes.LabelSource, "true"))
			defer internaltelemetry.ReportOnRecvPacketTelemetry(transferAmount, denomPath, labels)

			// Continue processing rest of tokens in packet data.
			continue
		}

		// sender chain is the source, mint vouchers

		// since SendPacket did not prefix the denomination, we must add the destination port and channel to the trace
		trace := []types.Trace{types.NewTrace(packet.DestinationPort, packet.DestinationChannel)}
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
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, receiver, sdk.NewCoins(voucher),
		); err != nil {
			return errorsmod.Wrapf(err, "failed to send coins to receiver %s", receiver.String())
		}

		denomPath := token.Denom.Path()
		labels = append(labels, telemetry.NewLabel(coretypes.LabelSource, "false"))
		defer internaltelemetry.ReportOnRecvPacketTelemetry(transferAmount, denomPath, labels)
	}

	return nil
}

// OnAcknowledgementPacket responds to the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement
// was a success then nothing occurs. If the acknowledgement failed, then
// the sender is refunded their tokens using the refundPacketTokens function.
func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketDataV2, ack channeltypes.Acknowledgement) error {
	switch ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		return nil
	case *channeltypes.Acknowledgement_Error:
		return k.refundPacketTokens(ctx, packet, data)
	default:
		return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected one of [%T, %T], got %T", channeltypes.Acknowledgement_Result{}, channeltypes.Acknowledgement_Error{}, ack.Response)
	}
}

// OnTimeoutPacket refunds the sender since the original packet sent was
// never received and has been timed out.
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketDataV2) error {
	return k.refundPacketTokens(ctx, packet, data)
}

// refundPacketTokens will unescrow and send back the tokens back to sender
// if the sending chain was the source chain. Otherwise, the sent tokens
// were burnt in the original send so new tokens are minted and sent to
// the sending address.
func (k Keeper) refundPacketTokens(ctx sdk.Context, packet channeltypes.Packet, data types.FungibleTokenPacketDataV2) error {
	// NOTE: packet data type already checked in handler.go

	for _, token := range data.Tokens {
		transferAmount, ok := sdkmath.NewIntFromString(token.Amount)
		if !ok {
			return errorsmod.Wrapf(types.ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", transferAmount)
		}

		coin := sdk.NewCoin(token.Denom.IBCDenom(), transferAmount)

		sender, err := sdk.AccAddressFromBech32(data.Sender)
		if err != nil {
			return err
		}

		if token.Denom.SenderChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel()) {
			// unescrow tokens back to sender
			escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
			if err := k.unescrowCoin(ctx, escrowAddress, sender, coin); err != nil {
				return err
			}

			// Continue processing rest of tokens in packet data.
			continue
		}

		// mint vouchers back to sender
		if err := k.bankKeeper.MintCoins(
			ctx, types.ModuleName, sdk.NewCoins(coin),
		); err != nil {
			return err
		}

		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sender, sdk.NewCoins(coin)); err != nil {
			panic(fmt.Errorf("unable to send coins from module to account despite previously minting coins to module account: %v", err))
		}
	}

	return nil
}

// escrowCoin will send the given coin from the provided sender to the escrow address. It will also
// update the total escrowed amount by adding the escrowed coin's amount to the current total escrow.
func (k Keeper) escrowCoin(ctx sdk.Context, sender, escrowAddress sdk.AccAddress, coin sdk.Coin) error {
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
func (k Keeper) unescrowCoin(ctx sdk.Context, escrowAddress, receiver sdk.AccAddress, coin sdk.Coin) error {
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
func (k Keeper) tokenFromCoin(ctx sdk.Context, coin sdk.Coin) (types.Token, error) {
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
func createPacketDataBytesFromVersion(appVersion, sender, receiver, memo string, tokens types.Tokens) []byte {
	var packetDataBytes []byte
	switch appVersion {
	case types.V1:
		// Sanity check, tokens must always be of length 1 if using app version V1.
		if len(tokens) != 1 {
			panic(fmt.Errorf("length of tokens must be equal to 1 if using %s version", types.V1))
		}

		token := tokens[0]
		packetData := types.NewFungibleTokenPacketData(token.Denom.Path(), token.Amount, sender, receiver, memo)
		packetDataBytes = packetData.GetBytes()
	case types.V2:
		packetData := types.NewFungibleTokenPacketDataV2(tokens, sender, receiver, memo)
		packetDataBytes = packetData.GetBytes()
	default:
		panic(fmt.Errorf("app version must be one of %s", types.SupportedVersions))
	}

	return packetDataBytes
}
