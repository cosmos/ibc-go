package types

import (
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
)

func (ifp *InFlightPacket) ChannelPacket() channeltypes.Packet {
	return channeltypes.Packet{
		Data:               ifp.PacketData,
		Sequence:           ifp.RefundSequence,
		SourcePort:         ifp.PacketSrcPortId,
		SourceChannel:      ifp.PacketSrcChannelId,
		DestinationPort:    ifp.RefundPortId,
		DestinationChannel: ifp.RefundChannelId,
		TimeoutHeight:      clienttypes.MustParseHeight(ifp.PacketTimeoutHeight),
		TimeoutTimestamp:   ifp.PacketTimeoutTimestamp,
	}
}
