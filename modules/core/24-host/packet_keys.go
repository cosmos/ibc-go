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
// NOTE: NextSequenceSendKey has been removed and we only use the IBC v2 key in this repo.
// We can safely do this since the NextSequenceSendKey is not proven to counterparties, thus we can use any key format we want.
// so long as they do not collide with other keys in the store.

// NextSequenceRecvKey returns the store key for the receive sequence of a particular
// channel binded to a specific port
func NextSequenceRecvKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s", KeyNextSeqRecvPrefix, ChannelPath(portID, channelID))
}

// NextSequenceAckKey returns the store key for the acknowledgement sequence of
// a particular channel binded to a specific port.
func NextSequenceAckKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s", KeyNextSeqAckPrefix, ChannelPath(portID, channelID))
}

// PacketCommitmentKey returns the store key of under which a packet commitment
// is stored
func PacketCommitmentKey(portID, channelID string, sequence uint64) []byte {
	return fmt.Appendf(nil, "%s/%d", PacketCommitmentPrefixKey(portID, channelID), sequence)
}

// PacketCommitmentPrefixKey defines the prefix for commitments to packet data fields store path.
func PacketCommitmentPrefixKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s/%s", KeyPacketCommitmentPrefix, ChannelPath(portID, channelID), KeySequencePrefix)
}

// PacketAcknowledgementKey returns the store key of under which a packet
// acknowledgement is stored
func PacketAcknowledgementKey(portID, channelID string, sequence uint64) []byte {
	return fmt.Appendf(nil, "%s/%d", PacketAcknowledgementPrefixKey(portID, channelID), sequence)
}

// PacketAcknowledgementPrefixKey defines the prefix for commitments to packet data fields store path.
func PacketAcknowledgementPrefixKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s/%s", KeyPacketAckPrefix, ChannelPath(portID, channelID), KeySequencePrefix)
}

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func PacketReceiptKey(portID, channelID string, sequence uint64) []byte {
	return fmt.Appendf(nil, "%s/%s/%s", KeyPacketReceiptPrefix, ChannelPath(portID, channelID), sequencePath(sequence))
}

// RecvStartSequenceKey returns the store key for the recv start sequence of a particular channel
func RecvStartSequenceKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s", KeyRecvStartSequence, ChannelPath(portID, channelID))
}

func sequencePath(sequence uint64) string {
	return fmt.Sprintf("%s/%d", KeySequencePrefix, sequence)
}
