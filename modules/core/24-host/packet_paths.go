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
	KeyPruningSequenceStart   = "pruningSequenceStart"
	KeyRecvStartSequence      = "recvStartSequence"
)

// ICS04
// The following paths are the keys to the store as defined in https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#store-paths

// NextSequenceSendPath defines the next send sequence counter store path
func NextSequenceSendPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyNextSeqSendPrefix, channelPath(portID, channelID))
}

// NextSequenceRecvPath defines the next receive sequence counter store path.
func NextSequenceRecvPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyNextSeqRecvPrefix, channelPath(portID, channelID))
}

// NextSequenceAckPath defines the next acknowledgement sequence counter store path
func NextSequenceAckPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyNextSeqAckPrefix, channelPath(portID, channelID))
}

// PacketCommitmentPath defines the commitments to packet data fields store path
func PacketCommitmentPath(portID, channelID string, sequence uint64) string {
	return fmt.Sprintf("%s/%d", PacketCommitmentPrefixPath(portID, channelID), sequence)
}

// PacketCommitmentPrefixPath defines the prefix for commitments to packet data fields store path.
func PacketCommitmentPrefixPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s", KeyPacketCommitmentPrefix, channelPath(portID, channelID), KeySequencePrefix)
}

// PacketAcknowledgementPath defines the packet acknowledgement store path
func PacketAcknowledgementPath(portID, channelID string, sequence uint64) string {
	return fmt.Sprintf("%s/%d", PacketAcknowledgementPrefixPath(portID, channelID), sequence)
}

// PacketAcknowledgementPrefixPath defines the prefix for commitments to packet data fields store path.
func PacketAcknowledgementPrefixPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s", KeyPacketAckPrefix, channelPath(portID, channelID), KeySequencePrefix)
}

// PacketReceiptPath defines the packet receipt store path
func PacketReceiptPath(portID, channelID string, sequence uint64) string {
	return fmt.Sprintf("%s/%s/%s", KeyPacketReceiptPrefix, channelPath(portID, channelID), sequencePath(sequence))
}

// PruningSequenceStartPath defines the path under which the pruning sequence starting value is stored
func PruningSequenceStartPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyPruningSequenceStart, channelPath(portID, channelID))
}

// RecvStartSequencePath defines the path under which the recv start sequence is stored
func RecvStartSequencePath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyRecvStartSequence, channelPath(portID, channelID))
}

func sequencePath(sequence uint64) string {
	return fmt.Sprintf("%s/%d", KeySequencePrefix, sequence)
}
