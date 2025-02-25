package host

import "fmt"

const (
	KeySequencePrefix         = "sequences"
	KeyNextSeqSendPrefix      = "nextSequenceSend"
	KeyNextSeqRecvPrefix      = "nextSequenceRecv"
	KeyNextSeqAckPrefix       = "nextSequenceAck"
	KeyPacketCommitmentPrefix = "commitments"
	KeyPacketAckPrefix        = "acks"
	KeyPacketReceiptPrefix    = "receipts"
	KeyRecvStartSequence      = "recvStartSequence"
)

// ICS04
// The following paths are the keys to the store as defined in https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#store-paths

// NextSequenceSendKey returns the store key for the send sequence of a particular
// channel binded to a specific port.
func NextSequenceSendKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyNextSeqSendPrefix, channelPath(portID, channelID)))
}

// NextSequenceRecvKey returns the store key for the receive sequence of a particular
// channel binded to a specific port
func NextSequenceRecvKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyNextSeqRecvPrefix, channelPath(portID, channelID)))
}

// NextSequenceAckKey returns the store key for the acknowledgement sequence of
// a particular channel binded to a specific port.
func NextSequenceAckKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyNextSeqAckPrefix, channelPath(portID, channelID)))
}

// PacketCommitmentKey returns the store key of under which a packet commitment
// is stored
func PacketCommitmentKey(portID, channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s/%d", PacketCommitmentPrefixKey(portID, channelID), sequence))
}

// PacketCommitmentPrefixKey defines the prefix for commitments to packet data fields store path.
func PacketCommitmentPrefixKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", KeyPacketCommitmentPrefix, channelPath(portID, channelID), KeySequencePrefix))
}

// PacketAcknowledgementKey returns the store key of under which a packet
// acknowledgement is stored
func PacketAcknowledgementKey(portID, channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s/%d", PacketAcknowledgementPrefixKey(portID, channelID), sequence))
}

// PacketAcknowledgementPrefixKey defines the prefix for commitments to packet data fields store path.
func PacketAcknowledgementPrefixKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", KeyPacketAckPrefix, channelPath(portID, channelID), KeySequencePrefix))
}

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func PacketReceiptKey(portID, channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", KeyPacketReceiptPrefix, channelPath(portID, channelID), sequencePath(sequence)))
}

// RecvStartSequenceKey returns the store key for the recv start sequence of a particular channel
func RecvStartSequenceKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyRecvStartSequence, channelPath(portID, channelID)))
}

func sequencePath(sequence uint64) string {
	return fmt.Sprintf("%s/%d", KeySequencePrefix, sequence)
}
