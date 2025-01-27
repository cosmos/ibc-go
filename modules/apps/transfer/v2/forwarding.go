package v2

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channelv2types "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
)

func (im *IBCModule) forwardPacket(ctx context.Context, destinationClient string, destinationPort string, sequence uint64, data types.FungibleTokenPacketDataV2, timeoutTimestamp uint64, receivedCoins sdk.Coins) error {
	var nextForwardingPath *types.Forwarding
	if len(data.Forwarding.Hops) > 1 {
		// remove the first hop since we are going to send to the first hop now and we want to propagate the rest of the hops to the receiver
		nextForwardingPath = types.NewForwarding(false, data.Forwarding.Hops[1:]...)
	}

	// sending from module account (used as a temporary forward escrow) to the original receiver address.
	sender := im.keeper.AuthKeeper.GetModuleAddress(types.ModuleName)

	if _, ok := im.chanV2Keeper.GetCounterparty(ctx, data.Forwarding.Hops[0].ChannelId); ok {
		tokens := make([]types.Token, len(receivedCoins))

		for i, coin := range receivedCoins {
			var err error
			tokens[i], err = im.keeper.TokenFromCoin(ctx, coin)
			if err != nil {
				return err
			}
		}
		packetData, err := keeper.CreatePacketDataBytesFromVersion(types.V2, sender.String(), data.Receiver, data.Memo, tokens, nextForwardingPath.Hops)
		if err != nil {
			return err
		}
		payload := channelv2types.Payload{
			SourcePort:      data.Forwarding.Hops[0].PortId,
			DestinationPort: data.Forwarding.Hops[0].PortId,
			Version:         types.V2,
			Encoding:        types.EncodingProtobuf,
			Value:           packetData,
		}
		// V2 counterparty exists for next hop so we will send a IBC V2 packet
		msg := channelv2types.NewMsgSendPacket(
			data.Forwarding.Hops[0].ChannelId,
			timeoutTimestamp,
			sender.String(),
			payload,
		)
		resp, err := im.chanV2Keeper.SendPacket(ctx, msg)
		if err != nil {
			return err
		}
		im.keeper.SetForwardV2PacketId(ctx, data.Forwarding.Hops[0].ChannelId, resp.Sequence, destinationClient, destinationPort, sequence)
	} else {
		// use v1 channel for forwarding
		msg := types.NewMsgTransfer(
			data.Forwarding.Hops[0].PortId,
			data.Forwarding.Hops[0].ChannelId,
			receivedCoins,
			sender.String(),
			data.Receiver,
			clienttypes.ZeroHeight(),
			timeoutTimestamp,
			data.Forwarding.DestinationMemo,
			nextForwardingPath,
		)

		resp, err := im.keeper.Transfer(ctx, msg)
		if err != nil {
			return err
		}
		im.keeper.SetForwardV2PacketId(ctx, data.Forwarding.Hops[0].ChannelId, resp.Sequence, destinationClient, destinationPort, sequence)
	}
	return nil
}

// revertForwardedPacket reverts the logic of receive packet that occurs in the middle chains during a packet forwarding.
// If the packet fails to be forwarded all the way to the final destination, the state changes on this chain must be reverted
// before sending back the error acknowledgement to ensure atomic packet forwarding.
func (im *IBCModule) revertForwardedPacket(ctx context.Context, awaitClient string, awaitPort string, awaitSequence uint64, failedPacketData types.FungibleTokenPacketDataV2) error {
	/*
		Recall that RecvPacket handles an incoming packet depending on the denom of the received funds:
			1. If the funds are native, then the amount is sent to the receiver from the escrow.
			2. If the funds are foreign, then a voucher token is minted.
		We revert it in this function by:
			1. Sending funds back to escrow if the funds are native.
			2. Burning voucher tokens if the funds are foreign
	*/

	forwardingAddr := im.keeper.AuthKeeper.GetModuleAddress(types.ModuleName)
	escrow := types.GetEscrowAddress(awaitPort, awaitClient)

	// we can iterate over the tokens we sent in the forwarding packet
	// to get the received tokens from the awaitPacket
	for _, token := range failedPacketData.Tokens {
		// parse the transfer amount
		coin, err := token.ToCoin()
		if err != nil {
			return err
		}

		// check if the token we received originated on the sender
		// given that the packet is being reversed, we check the DestinationChannel and DestinationPort
		// of the forwardedPacket to see if a hop was added to the trace during the receive step
		if token.Denom.HasPrefix(awaitPort, awaitClient) {
			if err := im.keeper.BankKeeper.BurnCoins(ctx, forwardingAddr, sdk.NewCoins(coin)); err != nil {
				return err
			}
		} else {
			// send it back to the escrow address
			if err := im.keeper.EscrowCoin(ctx, forwardingAddr, escrow, coin); err != nil {
				return err
			}
		}
	}
	return nil
}
