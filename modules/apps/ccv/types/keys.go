package types

const (
	// ModuleName defines the CCV module name
	ModuleName = "CCV"

	// Version defines the current version the IBC CCV parent and child
	// module supports
	Version = "1"

	// ChannelStatusKeyPrefix is the key prefix for storing the validation status of the CCV channel
	ChannelStatusKeyPrefix = "channelstatus"
)

// ChannelStatusKey returns the key under which the Status of a baby chain is stored.
func ChannelStatusKey(channelID string) []byte {
	return []byte(ChannelStatusKeyPrefix + "/" + channelID)
}
