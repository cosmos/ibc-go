package host

import "fmt"

const (
	KeyChannelEndPrefix     = "channelEnds"
	KeyChannelPrefix        = "channels"
	KeyChannelUpgradePrefix = "channelUpgrades"
	KeyChannelRestoreSuffix = "restore"
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

// ChannelRestorePath defines the path under which channel ends are stored for restoration in the event of upgrade handshake failure
func ChannelRestorePath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", channelUpgradePath(portID, channelID), KeyChannelRestoreSuffix)
}

// ChannelRestoreKey returns the store key for a particular channel end used for restoratoin in the event of upgrade handshake failure
func ChannelRestoreKey(portID, channelID string) []byte {
	return []byte(ChannelRestorePath(portID, channelID))
}

func channelPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", KeyPortPrefix, portID, KeyChannelPrefix, channelID)
}

func channelUpgradePath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyChannelUpgradePrefix, channelPath(portID, channelID))
}
