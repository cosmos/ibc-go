package types

const (
	// SubModuleName defines the IBC packet server name.
	SubModuleName = "packetserver"

	// ChannelKey is the key used to store channel in the client store.
	// the channel key is imported from types instead of host because
	// the channel key is not a part of the ics-24 host specification
	ChannelKey = "channel"
)
