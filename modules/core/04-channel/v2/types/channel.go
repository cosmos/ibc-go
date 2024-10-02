package types

func NewPacket(sequence uint64, sourceID, destinationID string, timeoutTimestamp uint64, data ...PacketData) Packet {
	return Packet{
		Sequence:         sequence,
		SourceId:         sourceID,
		DestinationId:    destinationID,
		TimeoutTimestamp: timeoutTimestamp,
		Data:             data,
	}
}

func (p Packet) ValidateBasic() error {
	// TODO: https://github.com/cosmos/ibc-go/issues/7385
	return nil
}
