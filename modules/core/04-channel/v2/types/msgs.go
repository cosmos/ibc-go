package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1types "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
)

var _ sdk.Msg = (*MsgSendPacket)(nil)

// NewMsgSendPacket constructs and returns a new MsgSendPacket.
func NewMsgSendPacket(sourceID string, timeoutTimestamp uint64, signer string, packetData ...v1types.PacketData) *MsgSendPacket {
	return &MsgSendPacket{
		SourceId:         sourceID,
		TimeoutTimestamp: timeoutTimestamp,
		PacketData:       packetData,
		Signer:           signer,
	}
}
