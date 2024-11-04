package v2

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func PacketReceiptKey(channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("receipts/channels/%s/sequences/%s", channelID, sdk.Uint64ToBigEndian(sequence)))
}

// PacketAcknowledgementKey returns the store key of under which a packet acknowledgement is stored.
func PacketAcknowledgementKey(channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("acks/channels/%s/sequences/%s", channelID, sdk.Uint64ToBigEndian(sequence)))
}

// PacketCommitmentPrefixKey returns the store key prefix under which packet commitments for a particular channel are stored.
func PacketCommitmentPrefixKey(channelID string) []byte {
	return []byte(fmt.Sprintf("commitments/channels/%s", channelID))
}

// PacketCommitmentKey returns the store key of under which a packet commitment is stored.
func PacketCommitmentKey(channelID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("commitments/channels/%s/sequences/%s", channelID, sdk.Uint64ToBigEndian(sequence)))
}

// NextSequenceSendKey returns the store key for the next sequence send of a given channelID.
func NextSequenceSendKey(channelID string) []byte {
	return []byte(fmt.Sprintf("nextSequenceSend/%s", channelID))
}
