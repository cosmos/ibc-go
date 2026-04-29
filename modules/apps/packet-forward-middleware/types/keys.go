package types

import "fmt"

const (
	// ModuleName defines the module name
	// NOTE: There is a spelling mistake in the module name that came from the original implementation
	// and is currently kept for backward compatibility. Consideration for renaming done in #8388
	ModuleName = "packetfowardmiddleware"

	// StoreKey is the store key string for IBC transfer
	StoreKey = ModuleName

	// QuerierRoute is the querier route for IBC transfer
	QuerierRoute = ModuleName

	ForwardMetadataKey = "forward"
	ForwardReceiverKey = "receiver"
	ForwardPortKey     = "port"
	ForwardChannelKey  = "channel"
	ForwardTimeoutKey  = "timeout"
	ForwardRetriesKey  = "retries"
	ForwardNextKey     = "next"
)

type NonrefundableKey struct{}

func RefundPacketKey(channelID, portID string, sequence uint64) []byte {
	return fmt.Appendf(nil, "%s/%s/%d", channelID, portID, sequence)
}
