package host

import "fmt"

const (
	KeyChannelEndPrefix = "channelEnds"
	KeyChannelPrefix    = "channels"
)

// ICS04
// The following paths are the keys to the store as defined in https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#store-paths

// ChannelKey returns the store key for a particular channel
func ChannelKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s", KeyChannelEndPrefix, channelPath(portID, channelID))
}

func channelPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", KeyPortPrefix, portID, KeyChannelPrefix, channelID)
}
