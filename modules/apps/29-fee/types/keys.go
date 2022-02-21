package types

import (
	"fmt"

	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
)

const (
	// ModuleName defines the 29-fee name
	ModuleName = "feeibc"

	// StoreKey is the store key string for IBC fee module
	StoreKey = ModuleName

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

	// FeesInEscrowPrefix is the key prefix for fee in escrow mapping
	FeesInEscrowPrefix = "feesInEscrow"

	// ForwardRelayerPrefix is the key prefix for forward relayer addresses stored in state for async acknowledgements
	ForwardRelayerPrefix = "forwardRelayer"

	AttributeKeyRecvFee    = "recv_fee"
	AttributeKeyAckFee     = "ack_fee"
	AttributeKeyTimeoutFee = "timeout_fee"
)

// FeeEnabledKey returns the key that stores a flag to determine if fee logic should
// be enabled for the given port and channel identifiers.
func FeeEnabledKey(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", FeeEnabledKeyPrefix, portID, channelID))
}

// KeyRelayerAddress returns the key for relayer address -> counteryparty address mapping
func KeyRelayerAddress(address, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", RelayerAddressKeyPrefix, address, channelID))
}

// KeyForwardRelayerAddress returns the key for packetID -> forwardAddress mapping
func KeyForwardRelayerAddress(packetId channeltypes.PacketId) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s/%d/", ForwardRelayerPrefix, packetId.PortId, packetId.ChannelId, packetId.Sequence))
}

// KeyFeeInEscrow returns the key for escrowed fees
func KeyFeeInEscrow(packetID channeltypes.PacketId) []byte {
	return []byte(fmt.Sprintf("%s/%d", KeyFeeInEscrowChannelPrefix(packetID.PortId, packetID.ChannelId), packetID.Sequence))
}

// KeyFeesInEscrow returns the key for escrowed fees
func KeyFeesInEscrow(packetID channeltypes.PacketId) []byte {
	return []byte(fmt.Sprintf("%s/%d", KeyFeesInEscrowChannelPrefix(packetID.PortId, packetID.ChannelId), packetID.Sequence))
}

// KeyFeeInEscrowChannelPrefix returns the key prefix for escrowed fees on the given channel
func KeyFeeInEscrowChannelPrefix(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s/packet", FeeInEscrowPrefix, portID, channelID))
}

// KeyFeesInEscrowChannelPrefix returns the key prefix for escrowed fees on the given channel
func KeyFeesInEscrowChannelPrefix(portID, channelID string) []byte {
	return []byte(fmt.Sprintf("%s/%s/%s", FeesInEscrowPrefix, portID, channelID))
}
