package host

import "fmt"

const (
	KeyChannelEndPrefix     = "channelEnds"
	KeyChannelPrefix        = "channels"
	KeyChannelUpgradePrefix = "channelUpgrades"
	KeyChannelRestorePrefix = "restore"
	KeyUpgradeError         = "upgradeError"
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
	return fmt.Sprintf("%s/%s/%s", KeyChannelUpgradePrefix, KeyChannelRestorePrefix, channelPath(portID, channelID))
}

// ChannelRestoreKey returns the store key for a particular channel end used for restoration in the event of upgrade handshake failure
func ChannelRestoreKey(portID, channelID string) []byte {
	return []byte(ChannelRestorePath(portID, channelID))
}

func channelPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", KeyPortPrefix, portID, KeyChannelPrefix, channelID)
}

// ErrorPath stores the ErrorReceipt in the case that a chain does not accept the proposed upgrade
func ErrorPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", KeyChannelUpgradePrefix, KeyPortPrefix, portID, KeyChannelPrefix, channelID, KeyUpgradeError)
}
