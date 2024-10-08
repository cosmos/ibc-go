package types

// NewMsgSendPacket creates a new MsgSendPacket instance.
func NewMsgSendPacket(sourceID string, timeoutTimestamp uint64, signer string, packetData ...PacketData) *MsgSendPacket {
	return &MsgSendPacket{
		SourceId:         sourceID,
		TimeoutTimestamp: timeoutTimestamp,
		PacketData:       packetData,
		Signer:           signer,
	}
}
