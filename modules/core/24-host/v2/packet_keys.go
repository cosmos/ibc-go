package v2

import "fmt"

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func PacketReceiptKey(sourceID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("receipts/channels/%s/sequences/%d", sourceID, sequence))
}

// PacketAcknowledgementKey returns the store key of under which a packet acknowledgement is stored.
func PacketAcknowledgementKey(sourceID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("acks/channels/%s/sequences/%d", sourceID, sequence))
}

// PacketCommitmentKey returns the store key of under which a packet commitment is stored.
func PacketCommitmentKey(sourceID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("commitments/channels/%s/sequences/%d", sourceID, sequence))
}

// NextSequenceSendKey returns the store key for the next sequence send of a given sourceID.
func NextSequenceSendKey(sourceID string) []byte {
	return []byte(fmt.Sprintf("nextSequenceSend/%s", sourceID))
}
