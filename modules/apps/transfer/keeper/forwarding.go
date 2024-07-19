package keeper

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

// forwardPacket forwards a fungible FungibleTokenPacketDataV2 to the next hop in the forwarding path.
func (k Keeper) forwardPacket(ctx sdk.Context, data types.FungibleTokenPacketDataV2, packet channeltypes.Packet, receivedCoins sdk.Coins) error {
	var nextForwardingPath *types.Forwarding
	if len(data.Forwarding.Hops) > 1 {
		// remove the first hop since we are going to send to the first hop now and we want to propagate the rest of the hops to the receiver
		nextForwardingPath = types.NewForwarding(false, data.Forwarding.Hops[1:]...)
	}

	// sending from module account (used as a temporary forward escrow) to the original receiver address.
	sender := k.authKeeper.GetModuleAddress(types.ModuleName)

	msg := types.NewMsgTransfer(
		data.Forwarding.Hops[0].PortId,
		data.Forwarding.Hops[0].ChannelId,
		receivedCoins,
		sender.String(),
		data.Receiver,
		clienttypes.ZeroHeight(),
		packet.TimeoutTimestamp,
		data.Forwarding.DestinationMemo,
		nextForwardingPath,
	)

	resp, err := k.Transfer(ctx, msg)
	if err != nil {
		return err
	}

	k.setForwardedPacket(ctx, data.Forwarding.Hops[0].PortId, data.Forwarding.Hops[0].ChannelId, resp.Sequence, packet)
	return nil
}

// acknowledgeForwardedPacket writes the async acknowledgement for packet
func (k Keeper) acknowledgeForwardedPacket(ctx sdk.Context, packet, forwardedPacket channeltypes.Packet, ack channeltypes.Acknowledgement) error {
	capability, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(packet.DestinationPort, packet.DestinationChannel))
	if !ok {
		return errorsmod.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	if err := k.ics4Wrapper.WriteAcknowledgement(ctx, capability, packet, ack); err != nil {
		return err
	}

	k.deleteForwardedPacket(ctx, forwardedPacket.SourcePort, forwardedPacket.SourceChannel, forwardedPacket.Sequence)
	return nil
}

// revertForwardedPacket reverts the logic of receive packet that occurs in the middle chains during a packet forwarding.
// If the packet fails to be forwarded all the way to the final destination, the state changes on this chain must be reverted
// before sending back the error acknowledgement to ensure atomic packet forwarding.
func (k Keeper) revertForwardedPacket(ctx sdk.Context, forwardedPacket channeltypes.Packet, failedPacketData types.FungibleTokenPacketDataV2) error {
	/*
		Recall that RecvPacket handles an incoming packet depending on the denom of the received funds:
			1. If the funds are native, then the amount is sent to the receiver from the escrow.
			2. If the funds are foreign, then a voucher token is minted.
		We revert it in this function by:
			1. Sending funds back to escrow if the funds are native.
			2. Burning voucher tokens if the funds are foreign
	*/

	forwardingAddr := k.authKeeper.GetModuleAddress(types.ModuleName)
	escrow := types.GetEscrowAddress(forwardedPacket.DestinationPort, forwardedPacket.DestinationChannel)

	// we can iterate over the received tokens of forwardedPacket by iterating over the sent tokens of failedPacketData
	for _, token := range failedPacketData.Tokens {
		// parse the transfer amount
		coin, err := token.ToCoin()
		if err != nil {
			return err
		}

		// check if the token we received originated on the sender
		// given that the packet is being reversed, we check the DestinationChannel and DestinationPort
		// of the forwardedPacket to see if a hop was added to the trace during the receive step
		if token.Denom.HasPrefix(forwardedPacket.DestinationPort, forwardedPacket.DestinationChannel) {
			if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
				return err
			}
		} else {
			// send it back to the escrow address
			if err := k.escrowCoin(ctx, forwardingAddr, escrow, coin); err != nil {
				return err
			}
		}
	}
	return nil
}

// getReceiverFromPacketData returns either the sender specified in the packet data or the forwarding address
// if there are still hops left to perform.
func (k Keeper) getReceiverFromPacketData(data types.FungibleTokenPacketDataV2) (sdk.AccAddress, error) {
	if data.HasForwarding() {
		// since data.Receiver can potentially be a non-CosmosSDK AccAddress, we return early if the packet should be forwarded
		return k.authKeeper.GetModuleAddress(types.ModuleName), nil
	}

	receiver, err := sdk.AccAddressFromBech32(data.Receiver)
	if err != nil {
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "failed to decode receiver address %s: %v", data.Receiver, err)
	}

	return receiver, nil
}
