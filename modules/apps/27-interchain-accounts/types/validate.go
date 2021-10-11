package types

import (
	"regexp"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DefaultMaxAddrLength defines the default maximum character length used in validation of addresses
var DefaultMaxAddrLength = 128

// IsValidAddr defines a regular expression to check if the provided string consists of
// strictly alphanumeric characters
var IsValidAddr = regexp.MustCompile("^[a-zA-Z0-9]*$").MatchString

// ValidateVersion performs basic validation of the provided ics27 version string.
// An ics27 version string may include an optional account address as per [TODO: Add spec when available]
// ValidateVersion first attempts to split the version string using the standard delimiter, then asserts a supported
// version prefix is included, followed by additional checks which enforce constraints on the account address.
func ValidateVersion(version string) error {
	s := strings.Split(version, Delimiter)

	if len(s) != 2 {
		return sdkerrors.Wrap(ErrInvalidVersion, "unexpected address format")
	}

	if s[0] != VersionPrefix {
		return sdkerrors.Wrapf(ErrInvalidVersion, "expected %s, got %s", VersionPrefix, s[0])
	}

	if !IsValidAddr(s[1]) || len(s[1]) == 0 || len(s[1]) > DefaultMaxAddrLength {
		return sdkerrors.Wrapf(
			ErrInvalidAccountAddress,
			"address must contain strictly alphanumeric characters, not exceeding %d characters in length",
			DefaultMaxAddrLength,
		)
	}

	return nil
}
