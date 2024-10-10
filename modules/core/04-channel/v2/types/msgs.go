package types

// NewMsgSendPacket creates a new MsgSendPacket instance.
func NewMsgSendPacket(sourceChannel string, timeoutTimestamp uint64, signer string, packetData ...PacketData) *MsgSendPacket {
	return &MsgSendPacket{
		SourceChannel:    sourceChannel,
		TimeoutTimestamp: timeoutTimestamp,
		PacketData:       packetData,
		Signer:           signer,
	}
}
