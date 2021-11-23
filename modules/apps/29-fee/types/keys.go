package types

import (
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
)

const (
	// ModuleName defines the 29-fee name
	ModuleName = "feeibc"

	// StoreKey is the store key string for IBC fee module
	StoreKey = ModuleName

	// PortKey is the port id that is wrapped by fee middleware
	PortID = "feetransfer"

	// RouterKey is the message route for IBC fee module
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC fee module
	QuerierRoute = ModuleName

	Version = "fee29-1"

	// FeeEnabledPrefix is the key prefix for storing fee enabled flag
	FeeEnabledKeyPrefix = "feeEnabled"

	// RelayerAddressKeyPrefix is the key prefix for relayer address mapping
	RelayerAddressKeyPrefix = "relayerAddress"

	// FeeInEscrowPrefix is the key prefix for fee in escrow mapping
	FeeInEscrowPrefix = "feeInEscrow"
)

// FeeEnabledKey returns the key that stores a flag to determine if fee logic should
// be enabled for the given port and channel identifiers.
func FeeEnabledKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", FeeEnabledKeyPrefix, portID, channelID))
}

// KeyRelayerAddress returns the key for relayer address -> counteryparty address mapping
func KeyRelayerAddress(address string) []byte {
	return []byte(fmt.Sprintf("%s/%s", RelayerAddressKeyPrefix, address))
}

// KeyFeeInEscrow returns the key for escrowed fees
func KeyFeeInEscrow(packetID *channeltypes.PacketId) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s/packet/%d", FeeInEscrowPrefix, packetID.PortId, packetID.ChannelId, packetID.Sequence))
}
