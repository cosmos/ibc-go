package types

const (
	// SubModuleName defines the channelv2 module name.
	SubModuleName = "channelv2"

	// ChannelKey is the key used to store channel in the client store.
	// the channel key is imported from types instead of host because
	// the channel key is not a part of the ics-24 host specification
	ChannelKey = "channel"

	// CreatorKey is the key used to store the client creator in the client store
	// the creator key is imported from types instead of host because
	// the creator key is not a part of the ics-24 host specification
	CreatorKey = "creator"
)
