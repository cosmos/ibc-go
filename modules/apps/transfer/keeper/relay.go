package keeper

import (
	"fmt"
	"strings"
	"time"

	metrics "github.com/hashicorp/go-metrics"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/internal/convert"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	v3types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types/v3"
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
	forwardingPath *types.ForwardingInfo,
) (uint64, error) {
	channel, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return 0, errorsmod.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
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

	var tokens []*v3types.Token

	for _, coin := range coins {
		// NOTE: denomination and hex hash correctness checked during msg.ValidateBasic
		fullDenomPath := coin.Denom

		var err error

		// deconstruct the token denomination into the denomination trace info
		// to determine if the sender is the source chain
		if strings.HasPrefix(coin.Denom, "ibc/") {
			fullDenomPath, err = k.DenomPathFromHash(ctx, coin.Denom)
			if err != nil {
				return 0, err
			}
		}

		// NOTE: SendTransfer simply sends the denomination as it exists on its own
		// chain inside the packet data. The receiving chain will perform denom
		// prefixing as necessary.

		if types.SenderChainIsSource(sourcePort, sourceChannel, fullDenomPath) {
			labels = append(labels, telemetry.NewLabel(coretypes.LabelSource, "true"))

			// obtain the escrow address for the source channel end
			escrowAddress := types.GetEscrowAddress(sourcePort, sourceChannel)
			if err := k.escrowToken(ctx, sender, escrowAddress, coin); err != nil {
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

		denom, trace := convert.ExtractDenomAndTraceFromV1Denom(fullDenomPath)
		token := &v3types.Token{
			Denom:  denom,
			Amount: coin.Amount.String(),
			Trace:  trace,
		}
		tokens = append(tokens, token)
	}

	// If we put the memo check in the validate basic, in case this line will return an error.
	packetData := v3types.NewFungibleTokenPacketData(tokens, sender.String(), receiver, memo, forwardingPath)

	sequence, err := k.ics4Wrapper.SendPacket(ctx, channelCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, packetData.GetBytes())
	if err != nil {
		return 0, err
	}

	defer func() {
		for _, token := range tokens {
			amount, ok := sdkmath.NewIntFromString(token.Amount)
			if ok && amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "ibc", "transfer"},
					float32(amount.Int64()),
					[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, token.GetFullDenomPath())},
				)
			}
		}

		telemetry.IncrCounterWithLabels(
			[]string{"ibc", types.ModuleName, "send"},
			1,
			labels,
		)
	}()

	return sequence, nil
}

// OnRecvPacket processes a cross chain fungible token transfer. If the
// sender chain is the source of minted tokens then vouchers will be minted
// and sent to the receiving address. Otherwise if the sender chain is sending
// back tokens this chain originally transferred to it, the tokens are
// unescrowed and sent to the receiving address.
func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, data v3types.FungibleTokenPacketData) (error, bool) {
	// validate packet data upon receiving
	// same things here if we put the validate memo here this function would return an error.

	if err := data.ValidateBasic(); err != nil {
		return errorsmod.Wrapf(err, "error validating ICS-20 transfer packet data"), false
	}

	if !k.GetParams(ctx).ReceiveEnabled {
		return types.ErrReceiveDisabled, false
	}

	// Receiver addresses logic:

	var finalReceiver sdk.AccAddress
	var receiver sdk.AccAddress
	var err error

	if data.ForwardingPath != nil && len(data.ForwardingPath.Hops) > 0 {
		// Transaction would abort already for previous check on Memo
		forwardAddress := types.GetForwardAddress(packet.DestinationPort, packet.DestinationChannel)
		receiver = forwardAddress
		finalReceiver, err = sdk.AccAddressFromBech32(data.Receiver)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to decode receiver address: %s", data.Receiver), false
		}
	} else {
		receiver, err = sdk.AccAddressFromBech32(data.Receiver)
		if err != nil {
			return errorsmod.Wrapf(err, "failed to decode receiver address: %s", data.Receiver), false
		}
	}

	if string(data.Sender) == "" {
		return errorsmod.Wrapf(err, "empty sender address: %s", data.Sender), false
	}

	if string(receiver) == "" {
		return errorsmod.Wrapf(err, "empty receiver address: %s", receiver), false
	}

	var receivedTokens sdk.Coins

	for _, token := range data.Tokens {
		fullDenomPath := token.GetFullDenomPath()

		labels := []metrics.Label{
			telemetry.NewLabel(coretypes.LabelSourcePort, packet.GetSourcePort()),
			telemetry.NewLabel(coretypes.LabelSourceChannel, packet.GetSourceChannel()),
		}

		// parse the transfer amount
		transferAmount, ok := sdkmath.NewIntFromString(token.Amount)
		if !ok {
			return errorsmod.Wrapf(types.ErrInvalidAmount, "unable to parse transfer amount: %s", token.Amount), false
		}

		// This is the prefix that would have been prefixed to the denomination
		// on sender chain IF and only if the token originally came from the
		// receiving chain.
		//
		// NOTE: We use SourcePort and SourceChannel here, because the counterparty
		// chain would have prefixed with DestPort and DestChannel when originally
		// receiving this coin as seen in the "sender chain is the source" condition.
		if types.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), fullDenomPath) {
			// sender chain is not the source, unescrow tokens

			// remove prefix added by sender chain
			voucherPrefix := types.GetDenomPrefix(packet.GetSourcePort(), packet.GetSourceChannel())
			unprefixedDenom := fullDenomPath[len(voucherPrefix):]

			// coin denomination used in sending from the escrow address
			denom := unprefixedDenom

			// The denomination used to send the coins is either the native denom or the hash of the path
			// if the denomination is not native.
			denomTrace := types.ParseDenomTrace(unprefixedDenom)
			if !denomTrace.IsNativeDenom() {
				denom = denomTrace.IBCDenom()
			}
			token := sdk.NewCoin(denom, transferAmount)

			if k.bankKeeper.BlockedAddr(receiver) {
				return errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "%s is not allowed to receive funds", receiver), false
			}

			escrowAddress := types.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
			if err := k.unescrowToken(ctx, escrowAddress, receiver, token); err != nil {
				return err, false
			}

			defer func() {
				if transferAmount.IsInt64() {
					telemetry.SetGaugeWithLabels(
						[]string{"ibc", types.ModuleName, "packet", "receive"},
						float32(transferAmount.Int64()),
						[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, unprefixedDenom)},
					)
				}

				telemetry.IncrCounterWithLabels(
					[]string{"ibc", types.ModuleName, "receive"},
					1,
					append(
						labels, telemetry.NewLabel(coretypes.LabelSource, "true"),
					),
				)
			}()
			// Appending token. The new denom has been computed
			receivedTokens = append(receivedTokens, token)

		}

		// sender chain is the source, mint vouchers

		// since SendPacket did not prefix the denomination, we must prefix denomination here
		prefixedDenom := types.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), fullDenomPath)

		// construct the denomination trace from the full raw denomination
		denomTrace := types.ParseDenomTrace(prefixedDenom)

		traceHash := denomTrace.Hash()
		if !k.HasDenomTrace(ctx, traceHash) {
			k.SetDenomTrace(ctx, denomTrace)
		}

		voucherDenom := denomTrace.IBCDenom()
		if !k.bankKeeper.HasDenomMetaData(ctx, voucherDenom) {
			k.setDenomMetadata(ctx, denomTrace)
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeDenomTrace,
				sdk.NewAttribute(types.AttributeKeyTraceHash, traceHash.String()),
				sdk.NewAttribute(types.AttributeKeyDenom, voucherDenom),
			),
		)
		voucher := sdk.NewCoin(voucherDenom, transferAmount)

		// mint new tokens if the source of the transfer is the same chain
		if err := k.bankKeeper.MintCoins(
			ctx, types.ModuleName, sdk.NewCoins(voucher),
		); err != nil {
			return errorsmod.Wrap(err, "failed to mint IBC tokens"), false
		}

		// send to receiver
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(
			ctx, types.ModuleName, receiver, sdk.NewCoins(voucher),
		); err != nil {
			return errorsmod.Wrapf(err, "failed to send coins to receiver %s", receiver.String()), false
		}

		defer func() {
			if transferAmount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"ibc", types.ModuleName, "packet", "receive"},
					float32(transferAmount.Int64()),
					[]metrics.Label{telemetry.NewLabel(coretypes.LabelDenom, fullDenomPath)},
				)
			}

			telemetry.IncrCounterWithLabels(
				[]string{"ibc", types.ModuleName, "receive"},
				1,
				append(
					labels, telemetry.NewLabel(coretypes.LabelSource, "false"),
				),
			)
		}()

		// Appending voucher. The new denom has been computed
		receivedTokens = append(receivedTokens, voucher)
	} // END OF FOR CYCLE

	/* If ack wasn't successful in the implementation we would have already errored out. No need of this check:
	if !ack.Success() {
		return ack
	  }
	*/

	// Adding forwarding logic
	if data.ForwardingPath != nil && len(data.ForwardingPath.Hops) > 0 {
		memo := ""
		nextForwardingPath := types.ForwardingInfo{
			Hops: data.ForwardingPath.Hops[1:],
			Memo: data.ForwardingPath.Memo,
		}

		if len(data.ForwardingPath.Hops) == 1 {
			memo = data.ForwardingPath.Memo
			nextForwardingPath = types.ForwardingInfo{
				Hops: nil,
				Memo: data.ForwardingPath.Memo,
			}
		}
		// Assign to timestamp --> current + 1 h
		timeoutTimestamp := uint64(ctx.BlockTime().Add(time.Hour).UnixNano())
		sequence, err := k.sendTransfer(ctx, data.ForwardingPath.Hops[0].PortID, data.ForwardingPath.Hops[0].ChannelId, receivedTokens, receiver, string(finalReceiver), clienttypes.Height{}, timeoutTimestamp, memo, &nextForwardingPath)
		if err != nil {
			return err, false
		}

		k.SetForwardedPacket(ctx, data.ForwardingPath.Hops[0].PortID, data.ForwardingPath.Hops[0].ChannelId, sequence, packet)

		// The return value will be used by the onRecvPacket in the ibc_module.go
		// If we return nil here, then we will write a successful sinc ack.
		// If we reutrn an error, then we will write a sinc error ack
		// The onRecvPacket in the ibc_module.go has as return value ibcexported.Acknowledgement
		// and it will return either an error ack or succesfull ack.
		// Thus to achieve the correct behaviour, we should be able to return a nil value in the ibc_module.go
		// onRecvPacket function.
		// return nil
		// Instead of returning simply nil, we will return a sentinel value that can be used to return nil in the ibc_module.go
		// onRecvPacket function.
		return nil, true

	}
	// The ibc_module.go module will return the proper ack.
	return nil, false
}

// OnAcknowledgementPacket responds to the success or failure of a packet
// acknowledgement written on the receiving chain. If the acknowledgement
// was a success then nothing occurs. If the acknowledgement failed, then
// the sender is refunded their tokens using the refundPacketToken function.
func (k Keeper) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, data v3types.FungibleTokenPacketData, ack channeltypes.Acknowledgement) error {
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(packet.SourcePort, packet.SourceChannel))
	if !ok {
		errorsmod.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	prevPacket, found := k.GetForwardedPacket(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)
	if found {
		switch ack.Response.(type) {
		case *channeltypes.Acknowledgement_Result:
			// the acknowledgement succeeded on the receiving chain so nothing
			// needs to be executed and no error needs to be returned
			// here the ibc_module.go will generate the sucss ack.

			// Figure Out how to do this

			FungibleTokenPacketAcknowledgement := channeltypes.NewResultAcknowledgement([]byte("forwarded packet succeeded")) // []byte{byte(1)})
			//				Error:  "forwarded packet succeeded",
			//				Result: true,
			//			}

			return k.ics4Wrapper.WriteAcknowledgement(ctx, channelCap, packet, FungibleTokenPacketAcknowledgement)

		case *channeltypes.Acknowledgement_Error:
			// the forwarded packet has failed, thus the funds have been refunded to the forwarding address.
			// we must revert the changes that came from successfully receiving the tokens on our chain
			// before propogating the error acknowledgement back to original sender chain

			// Should Check return value
			k.revertInFlightChanges(ctx, packet, prevPacket, data)
			// Figure Out how to do this

			FungibleTokenPacketAcknowledgement := channeltypes.NewErrorAcknowledgement(fmt.Errorf("forwarded packet failed"))
			/* channeltypes.Acknowledgement{
				channeltypes.Acknowledgement.Response.Error:  "forwarded packet failed",
				channeltypes.Acknowledgement.Response.Result: false,
			}*/
			// set the acknowledgement so that it can be verified on the other side
			return k.ics4Wrapper.WriteAcknowledgement(ctx, channelCap, packet, FungibleTokenPacketAcknowledgement)

		default:
			return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected one of [%T, %T], got %T", channeltypes.Acknowledgement_Result{}, channeltypes.Acknowledgement_Error{}, ack.Response)
		}
	} else {
		switch ack.Response.(type) {
		case *channeltypes.Acknowledgement_Error:
			return k.refundPacketToken(ctx, packet, data)
		}
	}
	return nil
}

// OnTimeoutPacket refunds the sender since the original packet sent was
// never received and has been timed out.
func (k Keeper) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, data v3types.FungibleTokenPacketData) error {
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(packet.SourcePort, packet.SourceChannel))
	if !ok {
		errorsmod.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	prevPacket, found := k.GetForwardedPacket(ctx, packet.SourcePort, packet.SourceChannel, packet.Sequence)
	if found {
		k.revertInFlightChanges(ctx, packet, prevPacket, data)

		FungibleTokenPacketAcknowledgement := channeltypes.NewErrorAcknowledgement(fmt.Errorf("forwarded packet failed"))

		/*FungibleTokenPacketAcknowledgement := channeltypes.Acknowledgement{
			channeltypes.Acknowledgement.Response.Error:  "forwarded packet failed",
			channeltypes.Acknowledgement.Response.Result: false,
		}*/

		return k.ics4Wrapper.WriteAcknowledgement(ctx, channelCap, packet, FungibleTokenPacketAcknowledgement)
	} else {
		return k.refundPacketToken(ctx, packet, data)
	}
}

// refundPacketToken will unescrow and send back the tokens back to sender
// if the sending chain was the source chain. Otherwise, the sent tokens
// were burnt in the original send so new tokens are minted and sent to
// the sending address.
// Probably data is not really necessary here.
func (k Keeper) refundPacketToken(ctx sdk.Context, packet channeltypes.Packet, data v3types.FungibleTokenPacketData) error {
	// NOTE: packet data type already checked in handler.go

	for _, token := range data.Tokens {
		fullDenomPath := token.GetFullDenomPath()

		// parse the denomination from the full denom path
		trace := types.ParseDenomTrace(fullDenomPath)

		// parse the transfer amount
		transferAmount, ok := sdkmath.NewIntFromString(token.Amount)
		if !ok {
			return errorsmod.Wrapf(types.ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", transferAmount)
		}
		token := sdk.NewCoin(trace.IBCDenom(), transferAmount)

		// decode the sender address
		sender, err := sdk.AccAddressFromBech32(data.Sender)
		if err != nil {
			return err
		}

		if types.SenderChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), fullDenomPath) {
			// unescrow tokens back to sender
			escrowAddress := types.GetEscrowAddress(packet.GetSourcePort(), packet.GetSourceChannel())
			return k.unescrowToken(ctx, escrowAddress, sender, token)
		}

		// mint vouchers back to sender
		if err := k.bankKeeper.MintCoins(
			ctx, types.ModuleName, sdk.NewCoins(token),
		); err != nil {
			return err
		}

		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sender, sdk.NewCoins(token)); err != nil {
			panic(fmt.Errorf("unable to send coins from module to account despite previously minting coins to module account: %v", err))
		}
	}

	return nil
}

func (k Keeper) revertInFlightChanges(ctx sdk.Context, sentPacket channeltypes.Packet, receivedPacket channeltypes.Packet, sentPacketData v3types.FungibleTokenPacketData) error {
	forwardEscrow := types.GetEscrowAddress(sentPacket.SourcePort, sentPacket.SourceChannel)
	reverseEscrow := types.GetEscrowAddress(receivedPacket.DestinationPort, receivedPacket.DestinationChannel)
	// the token on our chain is the token in the sentPacket

	// We could unmarshall the sentPacket data and use it in the for or use sentPacketData
	// Need to understand how to deal with this. The bank functions wants a coin and not a v3token type
	for _, token := range sentPacketData.Tokens {

		fullDenomPath := token.GetFullDenomPath()

		// parse the denomination from the full denom path
		trace := types.ParseDenomTrace(fullDenomPath)

		// parse the transfer amount
		transferAmount, ok := sdkmath.NewIntFromString(token.Amount)
		if !ok {
			return errorsmod.Wrapf(types.ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", transferAmount)
		}
		sdkToken := sdk.NewCoin(trace.IBCDenom(), transferAmount)

		// check if the packet we sent out was sending as source or not
		// in this case we escrowed the outgoing tokens
		if types.SenderChainIsSource(sentPacket.SourcePort, sentPacket.SourceChannel, fullDenomPath) {
			// check if the packet we received was a source token for our chain
			// Check if here should be ReceiverChainIsSource
			if types.SenderChainIsSource(receivedPacket.DestinationPort, receivedPacket.DestinationChannel, fullDenomPath) {
				// receive sent tokens from the received escrow to the forward escrow account
				// so we must send the tokens back from the forward escrow to the original received escrow account
				// To check if this is proper

				return k.unescrowToken(ctx, forwardEscrow, reverseEscrow, sdkToken)
			} else {
				// receive minted vouchers and sent to the forward escrow account
				// so we must remove the vouchers from the forward escrow account and burn them
				// Is this ok?
				if err := k.bankKeeper.BurnCoins(
					ctx, types.ModuleName, sdk.NewCoins(sdkToken),
				); err != nil {
					return err
				}
			}
		} else {
			// in this case we burned the vouchers of the outgoing packets
			// check if the packet we received was a source token for our chain
			// in this case, the tokens were unescrowed from the reverse escrow account
			if types.SenderChainIsSource(receivedPacket.DestinationPort, receivedPacket.DestinationChannel, token.Denom) {
				// in this case we must mint the burned vouchers and send them back to the escrow account
				// mint vouchers back to sender
				// Is this ok?
				if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdkToken)); err != nil {
					return err
				}

				if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, reverseEscrow, sdk.NewCoins(sdkToken)); err != nil {
					panic(fmt.Errorf("unable to send coins from module to account despite previously minting coins to module account: %v", err))
				}
			}

			// if it wasn't a source token on receive, then we simply had minted vouchers and burned them in the receive.
			// So no state changes were made, and thus no reversion is necessary
		}
	}
	return nil
}

// escrowToken will send the given token from the provided sender to the escrow address. It will also
// update the total escrowed amount by adding the escrowed token to the current total escrow.
func (k Keeper) escrowToken(ctx sdk.Context, sender, escrowAddress sdk.AccAddress, token sdk.Coin) error {
	if err := k.bankKeeper.SendCoins(ctx, sender, escrowAddress, sdk.NewCoins(token)); err != nil {
		// failure is expected for insufficient balances
		return err
	}

	// track the total amount in escrow keyed by denomination to allow for efficient iteration
	currentTotalEscrow := k.GetTotalEscrowForDenom(ctx, token.GetDenom())
	newTotalEscrow := currentTotalEscrow.Add(token)
	k.SetTotalEscrowForDenom(ctx, newTotalEscrow)

	return nil
}

// unescrowToken will send the given token from the escrow address to the provided receiver. It will also
// update the total escrow by deducting the unescrowed token from the current total escrow.
func (k Keeper) unescrowToken(ctx sdk.Context, escrowAddress, receiver sdk.AccAddress, token sdk.Coin) error {
	if err := k.bankKeeper.SendCoins(ctx, escrowAddress, receiver, sdk.NewCoins(token)); err != nil {
		// NOTE: this error is only expected to occur given an unexpected bug or a malicious
		// counterparty module. The bug may occur in bank or any part of the code that allows
		// the escrow address to be drained. A malicious counterparty module could drain the
		// escrow address by allowing more tokens to be sent back then were escrowed.
		return errorsmod.Wrap(err, "unable to unescrow tokens, this may be caused by a malicious counterparty module or a bug: please open an issue on counterparty module")
	}

	// track the total amount in escrow keyed by denomination to allow for efficient iteration
	currentTotalEscrow := k.GetTotalEscrowForDenom(ctx, token.GetDenom())
	newTotalEscrow := currentTotalEscrow.Sub(token)
	k.SetTotalEscrowForDenom(ctx, newTotalEscrow)

	return nil
}

// DenomPathFromHash returns the full denomination path prefix from an ibc denom with a hash
// component.
func (k Keeper) DenomPathFromHash(ctx sdk.Context, denom string) (string, error) {
	// trim the denomination prefix, by default "ibc/"
	hexHash := denom[len(types.DenomPrefix+"/"):]

	hash, err := types.ParseHexHash(hexHash)
	if err != nil {
		return "", errorsmod.Wrap(types.ErrInvalidDenomForTransfer, err.Error())
	}

	denomTrace, found := k.GetDenomTrace(ctx, hash)
	if !found {
		return "", errorsmod.Wrap(types.ErrTraceNotFound, hexHash)
	}

	fullDenomPath := denomTrace.GetFullDenomPath()
	return fullDenomPath, nil
}
