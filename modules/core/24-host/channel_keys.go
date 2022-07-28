package host

import "fmt"

const (
	KeyChannelEndPrefix     = "channelEnds"
	KeyChannelPrefix        = "channels"
	KeyChannelUpgradePrefix = "channelUpgrades"
	KeyUpgradeTimeout       = "upgradeTimeout"
)

// ICS04
// The following paths are the keys to the store as defined in https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#store-paths

// ChannelPath defines the path under which channels are stored
func ChannelPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyChannelEndPrefix, channelPath(portID, channelID))
}

// ChannelKey returns the store key for a particular channel
func ChannelKey(portID, channelID string) []byte {
	return []byte(ChannelPath(portID, channelID))
}

// ChannelCapabilityPath defines the path under which capability keys associated
// with a channel are stored
func ChannelCapabilityPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyChannelCapabilityPrefix, channelPath(portID, channelID))
}

// ChannelUpgradeTimeoutPath defines the path set by the upgrade initiator to determine when the UPGRADETRY step
// should timeout.
func ChannelUpgradeTimeoutPath(portID, channelId string) string {
	return fmt.Sprintf("%s/%s/%s", KeyChannelUpgradePrefix, channelPath(portID, channelId), KeyUpgradeTimeout)
}

func channelPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", KeyPortPrefix, portID, KeyChannelPrefix, channelID)
}
