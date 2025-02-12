package v2

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	PacketCommitmentBasePrefix      = byte(1)
	PacketReceiptBasePrefix         = byte(2)
	PacketAcknowledgementBasePrefix = byte(3)
)

// PacketCommitmentPrefixKey returns the store key prefix under which packet commitments for a particular channel are stored.
// channelID must be a generated identifier, not provided externally so key collisions are not possible.
func PacketCommitmentPrefixKey(channelID string) []byte {
	return append([]byte(channelID), PacketCommitmentBasePrefix)
}

// PacketCommitmentKey returns the store key of under which a packet commitment is stored.
// channelID must be a generated identifier, not provided externally so key collisions are not possible.
func PacketCommitmentKey(channelID string, sequence uint64) []byte {
	return append(PacketCommitmentPrefixKey(channelID), sdk.Uint64ToBigEndian(sequence)...)
}

// PacketReceiptPrefixKey returns the store key prefix under which packet receipts for a particular channel are stored.
// channelID must be a generated identifier, not provided externally so key collisions are not possible.
func PacketReceiptPrefixKey(channelID string) []byte {
	return append([]byte(channelID), PacketReceiptBasePrefix)
}

// PacketReceiptKey returns the store key of under which a packet receipt is stored.
// channelID must be a generated identifier, not provided externally so key collisions are not possible.
func PacketReceiptKey(channelID string, sequence uint64) []byte {
	return append(PacketReceiptPrefixKey(channelID), sdk.Uint64ToBigEndian(sequence)...)
}

// PacketAcknowledgementPrefixKey returns the store key prefix under which packet acknowledgements for a particular channel are stored.
// channelID must be a generated identifier, not provided externally so key collisions are not possible.
func PacketAcknowledgementPrefixKey(channelID string) []byte {
	return append([]byte(channelID), PacketAcknowledgementBasePrefix)
}

// PacketAcknowledgementKey returns the store key of under which a packet acknowledgement is stored.
// channelID must be a generated identifier, not provided externally so key collisions are not possible.
func PacketAcknowledgementKey(channelID string, sequence uint64) []byte {
	return append(PacketAcknowledgementPrefixKey(channelID), sdk.Uint64ToBigEndian(sequence)...)
}

// NextSequenceSendKey returns the store key for the next sequence send of a given channelID.
func NextSequenceSendKey(channelID string) []byte {
	return []byte(fmt.Sprintf("nextSequenceSend/%s", channelID))
}
