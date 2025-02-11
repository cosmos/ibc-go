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

	// AllowAllPacketDataKeys holds the string key that allows all memo strings in authz transfer messages
	AllowAllPacketDataKeys = "*"

	KeyTotalEscrowPrefix = "totalEscrowForDenom"

	ParamsKey = "params"

	// V1 defines first version of the IBC transfer module
	V1 = "ics20-1"

	// escrowAddressVersion should remain as ics20-1 to avoid the address changing.
	// this address has been reasoned about to avoid collisions with other addresses
	// https://github.com/cosmos/cosmos-sdk/issues/7737#issuecomment-735671951
	escrowAddressVersion = V1
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
	// DenomTraceKey defines the key to store the denomination trace info in store
	DenomTraceKey = []byte{0x02}
	// DenomKey defines the key to store the token denomination in store
	DenomKey = []byte{0x03}

	// SupportedVersions defines all versions that are supported by the module
	SupportedVersions = []string{V1}
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

// TotalEscrowForDenomKey returns the store key of under which the total amount of
// source chain tokens in escrow is stored.
func TotalEscrowForDenomKey(denom string) []byte {
	return []byte(fmt.Sprintf("%s/%s", KeyTotalEscrowPrefix, denom))
}
