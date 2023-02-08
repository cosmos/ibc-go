package types

import (
	"crypto/sha256"
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the IBC transfer name
	ModuleName = "transfer"

	// Version defines the current version the IBC tranfer
	// module supports
	Version = "ics20-1"

	// PortID is the default port id that transfer module binds to
	PortID = "transfer"

	// StoreKey is the store key string for IBC transfer
	StoreKey = ModuleName

	// RouterKey is the message route for IBC transfer
	RouterKey = ModuleName

	// QuerierRoute is the querier route for IBC transfer
	QuerierRoute = ModuleName

	// DenomPrefix is the prefix used for internal SDK coin representation.
	DenomPrefix = "ibc"
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
	// DenomTraceKey defines the key to store the denomination trace info in store
	DenomTraceKey = []byte{0x02}
	// ChainToTupleKeyPrefix defines the key to store the chain to tuple info in store
	ChainToTupleKeyPrefix = []byte{0x03}
	// TupleToChainKeyPrefix defines the key to store the tuple to chain info in store
	TupleToChainKeyPrefix = []byte{0x04}
)

// GetChainToTupleKey returns the key used to store the chain to tuple info in store
func GetChainToTupleKey(chainName string) []byte {
	return append(ChainToTupleKeyPrefix, []byte(chainName)...)
}

// GetTupleToChainKey returns the key used to store the tuple to chain info in store
func GetTupleToChainKey(sourcePort, sourceChannel string) []byte {
	return append(TupleToChainKeyPrefix, []byte(fmt.Sprintf("%s/%s", sourcePort, sourceChannel))...)
}

// TupleToBytes converts a source port and source channel to a byte array
func TupleToBytes(sourcePort, sourceChannel string) []byte {
	return []byte(fmt.Sprintf("%s/%s", sourcePort, sourceChannel))
}

// GetChannelPortFromTuple returns the source port from the tuple
func GetChannelPortFromTuple(tupleBytes []byte) (string, string, error) {
	channelPortTuple := string(tupleBytes)
	split := strings.Split(channelPortTuple, "/")
	if len(split) != 2 {
		return "", "", fmt.Errorf("invalid port/channel key: %s", channelPortTuple)
	}
	return split[0], split[1], nil
}

// GetEscrowAddress returns the escrow address for the specified channel.
// The escrow address follows the format as outlined in ADR 028:
// https://github.com/cosmos/cosmos-sdk/blob/master/docs/architecture/adr-028-public-key-addresses.md
func GetEscrowAddress(portID, channelID string) sdk.AccAddress {
	// a slash is used to create domain separation between port and channel identifiers to
	// prevent address collisions between escrow addresses created for different channels
	contents := fmt.Sprintf("%s/%s", portID, channelID)

	// ADR 028 AddressHash construction
	preImage := []byte(Version)
	preImage = append(preImage, 0)
	preImage = append(preImage, contents...)
	hash := sha256.Sum256(preImage)
	return hash[:20]
}
