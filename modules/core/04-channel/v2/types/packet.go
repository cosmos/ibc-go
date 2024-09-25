package types

import (
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

func NewPacketV2(
	sequence uint64, sourceId,
	destinationId string,
	timeoutTimestamp uint64,
	data ...channeltypes.PacketData,
) channeltypes.PacketV2 {
	return channeltypes.PacketV2{
		Sequence:         sequence,
		SourceId:         sourceId,
		DestinationId:    destinationId,
		TimeoutTimestamp: timeoutTimestamp,
		Data:             data,
	}
}
