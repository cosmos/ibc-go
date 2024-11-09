package v2

import (
	"fmt"
)

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func PacketReceiptKey(channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("receipts/channels/%s/sequences/%d", channelID, sequence))
}

// PacketAcknowledgementKey returns the store key of under which a packet acknowledgement is stored.
func PacketAcknowledgementKey(channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("acks/channels/%s/sequences/%d", channelID, sequence))
}

// PacketCommitmentKey returns the store key of under which a packet commitment is stored.
func PacketCommitmentKey(channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("commitments/channels/%s/sequences/%d", channelID, sequence))
}

// NextSequenceSendKey returns the store key for the next sequence send of a given channelID.
func NextSequenceSendKey(channelID string) []byte {
	return []byte(fmt.Sprintf("nextSequenceSend/%s", channelID))
}
