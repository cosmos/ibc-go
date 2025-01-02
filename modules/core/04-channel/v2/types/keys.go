package types

const (
	// SubModuleName defines the channelv2 module name.
	SubModuleName = "channelv2"

	// ChannelPrefix is the prefix under which all v2 channels are stored.
	// It is imported from types since it is not part of the ics-24 host
	// specification.
	ChannelPrefix = "channels"

	// CreatorPrefix is the prefix under which all v2 channel creators are stored.
	// It is imported from types since it is not part of the ics-24 host
	// specification.
	CreatorPrefix = "creators"
)
