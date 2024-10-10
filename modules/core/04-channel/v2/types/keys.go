package types

const (
	// SubModuleName defines the channelv2 module name.
	SubModuleName = "channelv2"

	// ChannelKey is the key used to store channel in the client store.
	// the channel key is imported from types instead of host because
	// the channel key is not a part of the ics-24 host specification
	ChannelKey = "channel"
)
