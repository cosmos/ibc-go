package types

const (
	// SubModuleName defines the channelv2 module name.
	SubModuleName = "channelv2"

	// ChannelKey is the key used to store channels in the channel store.
	// the channel key is imported from types instead of host because
	// the channel key is not a part of the ics-24 host specification
	ChannelKey = "channel"

	// CreatorKey is the key used to store the channel creator in the channel store
	// the creator key is imported from types instead of host because
	// the creator key is not a part of the ics-24 host specification
	CreatorKey = "creator"
)

// PacketCommitmentPrefixKey returns the store key prefix under which packet commitments for a particular channel are stored.
func PacketCommitmentPrefixKey(channelID string) []byte {
	return append([]byte(channelID), byte(1))
}

// PacketAcknowledgementPrefixKey returns the store key prefix under which packet acknowledgements for a particular channel are stored.
func PacketAcknowledgementPrefixKey(channelID string) []byte {
	return append([]byte(channelID), byte(3))
}
