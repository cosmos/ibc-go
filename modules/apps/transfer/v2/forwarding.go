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
	// remove the first hop since we are going to send to the first hop now and we want to propagate the rest of the hops to the receiver
	nextForwardingPath := types.NewForwarding(false, data.Forwarding.Hops[1:]...)

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
