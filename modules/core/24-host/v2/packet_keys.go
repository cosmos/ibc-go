package v2

import "fmt"

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func PacketReceiptKey(sourceID string, bigEndianSequence []byte) []byte {
	return []byte(fmt.Sprintf("receipts/channels/%s/sequences/%s", sourceID, string(bigEndianSequence)))
}

// PacketAcknowledgementKey returns the store key of under which a packet acknowledgement is stored.
func PacketAcknowledgementKey(sourceID string, bigEndianSequence []byte) []byte {
	return []byte(fmt.Sprintf("acks/channels/%s/sequences/%s", sourceID, string(bigEndianSequence)))
}

// PacketCommitmentKey returns the store key of under which a packet commitment is stored.
func PacketCommitmentKey(sourceID string, bigEndianSequence []byte) []byte {
	return []byte(fmt.Sprintf("commitments/channels/%s/sequences/%s", sourceID, string(bigEndianSequence)))
}

// NextSequenceSendKey returns the store key for the next sequence send of a given sourceID.
func NextSequenceSendKey(sourceID string) []byte {
	return []byte(fmt.Sprintf("nextSequenceSend/%s", sourceID))
}
