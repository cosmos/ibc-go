package host

// NextSequenceSendKey returns the store key for the send sequence of a particular
// channel binded to a specific port.
func NextSequenceSendKey(portID, channelID string) []byte {
	return []byte(NextSequenceSendPath(portID, channelID))
}

// NextSequenceRecvKey returns the store key for the receive sequence of a particular
// channel binded to a specific port
func NextSequenceRecvKey(portID, channelID string) []byte {
	return []byte(NextSequenceRecvPath(portID, channelID))
}

// NextSequenceAckKey returns the store key for the acknowledgement sequence of
// a particular channel binded to a specific port.
func NextSequenceAckKey(portID, channelID string) []byte {
	return []byte(NextSequenceAckPath(portID, channelID))
}

// PacketCommitmentKey returns the store key of under which a packet commitment
// is stored
func PacketCommitmentKey(portID, channelID string, sequence uint64) []byte {
	return []byte(PacketCommitmentPath(portID, channelID, sequence))
}

// PacketAcknowledgementKey returns the store key of under which a packet
// acknowledgement is stored
func PacketAcknowledgementKey(portID, channelID string, sequence uint64) []byte {
	return []byte(PacketAcknowledgementPath(portID, channelID, sequence))
}

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func PacketReceiptKey(portID, channelID string, sequence uint64) []byte {
	return []byte(PacketReceiptPath(portID, channelID, sequence))
}

// PruningSequenceStartKey returns the store key for the pruning sequence start of a particular channel
func PruningSequenceStartKey(portID, channelID string) []byte {
	return []byte(PruningSequenceStartPath(portID, channelID))
}

// RecvStartSequenceKey returns the store key for the recv start sequence of a particular channel
func RecvStartSequenceKey(portID, channelID string) []byte {
	return []byte(RecvStartSequencePath(portID, channelID))
}
