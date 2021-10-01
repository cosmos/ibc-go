package types

import (
	"fmt"
	"strconv"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	porttypes "github.com/cosmos/ibc-go/v2/modules/core/05-port/types"
)

const (
	// ModuleName defines the interchain accounts module name
	ModuleName = "interchainaccounts"

	// VersionPrefix defines the current version for interchain accounts
	VersionPrefix = "ics27-1"

	// PortID is the default port id that the interchain accounts module binds to
	PortID = "ibcaccount"

	// StoreKey is the store key string for interchain accounts
	StoreKey = ModuleName

	// RouterKey is the message route for interchain accounts
	RouterKey = ModuleName

	// QuerierRoute is the querier route for interchain accounts
	QuerierRoute = ModuleName

	// Delimiter is the delimiter used for the interchain accounts version string
	Delimiter = "."
)

var (
	// PortKey defines the key to store the port ID in store
	PortKey = []byte{0x01}
)

// NewVersion returns a complete version string in the format: VersionPrefix + Delimter + AccAddress
func NewAppVersion(versionPrefix, accAddr string) string {
	return fmt.Sprint(versionPrefix, Delimiter, accAddr)
}

// KeyActiveChannel creates and returns a new key used for active channels store operations
func KeyActiveChannel(portID string) []byte {
	return []byte(fmt.Sprintf("activeChannel/%s", portID))
}

// KeyOwnerAccount creates and returns a new key used for owner account store operations
func KeyOwnerAccount(portID string) []byte {
	return []byte(fmt.Sprintf("owner/%s", portID))
}

// ParseControllerConnSequence attempts to parse the controller connection sequence from the provided port identifier
// The port identifier must match the controller chain format outlined in (TODO: link spec), otherwise an empty string is returned
func ParseControllerConnSequence(portID string) (uint64, error) {
	s := strings.Split(portID, Delimiter)
	if len(s) != 4 {
		return 0, sdkerrors.Wrap(porttypes.ErrInvalidPort, "failed to parse port identifier")
	}

	seq, err := strconv.ParseUint(s[1], 10, 64)
	if err != nil {
		return 0, sdkerrors.Wrapf(err, "failed to parse connection sequence (%s)", s[1])
	}

	return seq, nil
}

// ParseHostConnSequence attempts to parse the host connection sequence from the provided port identifier
// The port identifier must match the controller chain format outlined in (TODO: link spec), otherwise an empty string is returned
func ParseHostConnSequence(portID string) (uint64, error) {
	s := strings.Split(portID, Delimiter)
	if len(s) != 4 {
		return 0, sdkerrors.Wrap(porttypes.ErrInvalidPort, "failed to parse port identifier")
	}

	seq, err := strconv.ParseUint(s[2], 10, 64)
	if err != nil {
		return 0, sdkerrors.Wrapf(err, "failed to parse connection sequence (%s)", s[1])
	}

	return seq, nil
}

// ParseAddressFromVersion attempts to extract the associated account address from the provided version string
func ParseAddressFromVersion(version string) (string, error) {
	s := strings.Split(version, Delimiter)
	if len(s) != 2 {
		return "", sdkerrors.Wrap(ErrInvalidVersion, "failed to parse version")
	}

	return s[1], nil
}
