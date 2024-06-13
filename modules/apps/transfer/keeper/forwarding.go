package keeper

import (
	"errors"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

// ackForwardPacketError reverts the receive packet logic that occurs in the middle chain and writes the async ack for the prevPacket
func (k Keeper) ackForwardPacketError(ctx sdk.Context, prevPacket channeltypes.Packet, failedPacketData types.FungibleTokenPacketDataV2) error {
	// the forwarded packet has failed, thus the funds have been refunded to the intermediate address.
	// we must revert the changes that came from successfully receiving the tokens on our chain
	// before propagating the error acknowledgement back to original sender chain
	if err := k.revertForwardedPacket(ctx, prevPacket, failedPacketData); err != nil {
		return err
	}

	forwardAck := channeltypes.NewErrorAcknowledgement(errors.New("forwarded packet failed"))
	return k.acknowledgeForwardedPacket(ctx, prevPacket, forwardAck)
}

// ackForwardPacketSuccess writes a successful async acknowledgement for the prevPacket
func (k Keeper) ackForwardPacketSuccess(ctx sdk.Context, prevPacket channeltypes.Packet) error {
	forwardAck := channeltypes.NewResultAcknowledgement([]byte("forwarded packet succeeded"))
	return k.acknowledgeForwardedPacket(ctx, prevPacket, forwardAck)
}

// ackForwardPacketTimeout reverts the receive packet logic that occurs in the middle chain and writes a failed async ack for the prevPacket
func (k Keeper) ackForwardPacketTimeout(ctx sdk.Context, prevPacket channeltypes.Packet, timeoutPacketData types.FungibleTokenPacketDataV2) error {
	if err := k.revertForwardedPacket(ctx, prevPacket, timeoutPacketData); err != nil {
		return err
	}

	forwardAck := channeltypes.NewErrorAcknowledgement(errors.New("forwarded packet timed out"))
	return k.acknowledgeForwardedPacket(ctx, prevPacket, forwardAck)
}

// acknowledgeForwardedPacket writes the async acknowledgement for packet
func (k Keeper) acknowledgeForwardedPacket(ctx sdk.Context, packet channeltypes.Packet, ack channeltypes.Acknowledgement) error {
	capability, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(packet.DestinationPort, packet.DestinationChannel))
	if !ok {
		return errorsmod.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	return k.ics4Wrapper.WriteAcknowledgement(ctx, capability, packet, ack)
}

// revertForwardedPacket reverts the logic of receive packet that occurs in the middle chains during a packet forwarding.
// If the packet fails to be forwarded all the way to the final destination, the state changes on this chain must be reverted
// before sending back the error acknowledgement to ensure atomic packet forwarding.
func (k Keeper) revertForwardedPacket(ctx sdk.Context, prevPacket channeltypes.Packet, failedPacketData types.FungibleTokenPacketDataV2) error {
	/*
		Recall that RecvPacket handles an incoming packet depending on the denom of the received funds:
			1. If the funds are native, then the amount is sent to the receiver from the escrow.
			2. If the funds are foreign, then a voucher token is minted.
		We revert it in this function by:
			1. Sending funds back to escrow if the funds are native.
			2. Burning voucher tokens if the funds are foreign
	*/

	forwardingAddr := types.GetForwardAddress(prevPacket.DestinationPort, prevPacket.DestinationChannel)
	escrow := types.GetEscrowAddress(prevPacket.DestinationPort, prevPacket.DestinationChannel)

	// we can iterate over the received tokens of prevPacket by iterating over the sent tokens of failedPacketData
	for _, token := range failedPacketData.Tokens {
		// parse the transfer amount
		transferAmount, ok := sdkmath.NewIntFromString(token.Amount)
		if !ok {
			return errorsmod.Wrapf(types.ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", transferAmount)
		}
		coin := sdk.NewCoin(token.Denom.IBCDenom(), transferAmount)

		// check if the token we received originated on the sender
		// given that the packet is being reversed, we check the DestinationChannel and DestinationPort
		// of the prevPacket to see if a hop was added to the trace during the receive step
		if token.Denom.SenderChainIsSource(prevPacket.DestinationPort, prevPacket.DestinationChannel) {
			// then send it back to the escrow address
			if err := k.escrowCoin(ctx, forwardingAddr, escrow, coin); err != nil {
				return err
			}

			continue
		}

		if err := k.burnCoin(ctx, forwardingAddr, coin); err != nil {
			return err
		}
	}
	return nil
}

// forwardPacket forwards a fungible FungibleTokenPacketDataV2 to the next hop in the forwarding path.
func (k Keeper) forwardPacket(ctx sdk.Context, data types.FungibleTokenPacketDataV2, packet channeltypes.Packet, receivedCoins sdk.Coins) error {
	var memo string

	var nextForwardingPath *types.Forwarding
	if len(data.Forwarding.Hops) == 1 {
		memo = data.Forwarding.Memo
		nextForwardingPath = nil
	} else {
		nextForwardingPath = &types.Forwarding{
			Hops: data.Forwarding.Hops[1:],
			Memo: data.Forwarding.Memo,
		}
	}

	// sending from the forward escrow address to the original receiver address.
	sender := types.GetForwardAddress(packet.DestinationPort, packet.DestinationChannel)

	msg := types.NewMsgTransfer(
		data.Forwarding.Hops[0].PortId,
		data.Forwarding.Hops[0].ChannelId,
		receivedCoins,
		sender.String(),
		data.Receiver,
		packet.TimeoutHeight,
		packet.TimeoutTimestamp,
		memo,
		nextForwardingPath,
	)

	resp, err := k.Transfer(ctx, msg)
	if err != nil {
		return err
	}

	k.SetForwardedPacket(ctx, data.Forwarding.Hops[0].PortId, data.Forwarding.Hops[0].ChannelId, resp.Sequence, packet)
	return nil
}

// getReceiverFromPacketData returns either the sender specified in the packet data or the forwarding address
// if there are still hops left to perform.
func getReceiverFromPacketData(data types.FungibleTokenPacketDataV2, portID, channelID string) (sdk.AccAddress, error) {
	receiver, err := sdk.AccAddressFromBech32(data.Receiver)
	if err != nil {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "failed to decode receiver address %s: %v", data.Receiver, err)
	}

	if data.ShouldBeForwarded() {
		receiver = types.GetForwardAddress(portID, channelID)
	}

	return receiver, nil
}
