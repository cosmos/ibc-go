package types

import (
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the IBC transfer name
	ModuleName = "transfer"

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

	// AllowAllPacketDataKeys holds the string key that allows all packet data keys in authz transfer messages
	AllowAllPacketDataKeys = "*"

	KeyTotalEscrowPrefix  = "totalEscrowForDenom"
	KeyTotalForwardPrefix = "totalForwardForDenom"

	ParamsKey = "params"

	KeyPacketForwardPrefix = "forwardedPacket"
)

const (
	// Version defines the current version the IBC transfer
	// module supports
	Version = "ics20-2"

	// Version1 defines first version of the IBC transfer module
	Version1 = "ics20-1"

	// escrowAddressVersion should remain as ics20-1 to avoid the address changing.
	escrowAddressVersion = "ics20-1"

	// this new address is introduced specifically with ics20-2.
	forwardAddressVersion = "forwardingAddress"
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
	// DenomTraceKey defines the key to store the denomination trace info in store
	DenomTraceKey = []byte{0x02}

	// SupportedVersions defines all versions that are supported by the module
	SupportedVersions = []string{Version, Version1}
)

// GetEscrowAddress returns the escrow address for the specified channel.
// The escrow address follows the format as outlined in ADR 028:
// https://github.com/cosmos/cosmos-sdk/blob/master/docs/architecture/adr-028-public-key-addresses.md
func GetEscrowAddress(portID, channelID string) sdk.AccAddress {
	// a slash is used to create domain separation between port and channel identifiers to
	// prevent address collisions between escrow addresses created for different channels
	contents := fmt.Sprintf("%s/%s", portID, channelID)

	// ADR 028 AddressHash construction
	preImage := []byte(escrowAddressVersion)
	preImage = append(preImage, 0)
	preImage = append(preImage, contents...)
	hash := sha256.Sum256(preImage)
	return hash[:20]
}

// GetForwardAddress returns the escrow address for the specified channel.
func GetForwardAddress(portID, channelID string) sdk.AccAddress {
	// a slash is used to create domain separation between port and channel identifiers to
	// prevent address collisions between escrow addresses created for different channels
	contents := fmt.Sprintf("%s/%s", portID, channelID)

	// ADR 028 AddressHash construction
	preImage := []byte(forwardAddressVersion)
	preImage = append(preImage, 0)
	preImage = append(preImage, contents...)
	hash := sha256.Sum256(preImage)
	return hash[:20]
}

// TotalEscrowForDenomKey returns the store key of under which the total amount of
// source chain tokens in escrow is stored.
func TotalEscrowForDenomKey(denom string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyTotalEscrowPrefix, denom))
}

func TotalForwardForDenomKey(denom string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyTotalForwardPrefix, denom))
}

// PacketForwardPath returns the packet forward path as a byte slice for the provided portID, channelID, and nextPacketSequence
// func PacketForwardPath(portID, channelID string, nextPacketSequence uint64) []byte {
func PacketForwardPath(portID, channelID string) []byte {
	path := fmt.Sprintf("%s/%s/%s", KeyPacketForwardPrefix, portID, channelID) //, nextPacketSequence)
	return []byte(path)
}
