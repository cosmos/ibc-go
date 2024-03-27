package host

// ChannelKey returns the store key for a particular channel
func ChannelKey(portID, channelID string) []byte {
	return []byte(ChannelPath(portID, channelID))
}

// ChannelUpgradeErrorKey returns the store key for a particular channelEnd used to stor the ErrorReceipt in the case that a chain does not accept the proposed upgrade
func ChannelUpgradeErrorKey(portID, channelID string) []byte {
	return []byte(ChannelUpgradeErrorPath(portID, channelID))
}

// ChannelUpgradeKey returns the store key for a particular channel upgrade attempt
func ChannelUpgradeKey(portID, channelID string) []byte {
	return []byte(ChannelUpgradePath(portID, channelID))
}

// ChannelCounterpartyUpgradeKey returns the store key for the upgrade used on the counterparty channel.
func ChannelCounterpartyUpgradeKey(portID, channelID string) []byte {
	return []byte(ChannelCounterpartyUpgradePath(portID, channelID))
}
