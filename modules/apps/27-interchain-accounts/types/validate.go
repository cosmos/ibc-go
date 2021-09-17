package types

import (
	"regexp"
	"strings"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DefaultMaxAddrLength defines the default maximum character length used in validation of addresses
var DefaultMaxAddrLength = 64

// DefaultMinAddrLength defines the default minimum character length used in validation of addresses
var DefaultMinAddrLength = 32

// IsValidAddr defines a regular expression to check if the provided string consists of
// strictly alphanumeric characters
var IsValidAddr = regexp.MustCompile("^[a-zA-Z0-9]*$").MatchString

// ValidateVersion performs basic validation of the provided ics27 version string
// When no delimiter is present it compares against the expected version
func ValidateVersion(version string) error {
	s := strings.Split(version, Delimiter)

	if s[0] != Version {
		return sdkerrors.Wrapf(ErrInvalidVersion, "invalid version: expected %s, got %s", Version, s[0])
	}

	if len(s) > 1 {
		if !IsValidAddr(s[1]) || len(s[1]) > DefaultMaxAddrLength || len(s[1]) < DefaultMinAddrLength {
			return sdkerrors.Wrapf(
				ErrInvalidAccountAddress,
				"address must contain strictly alphanumeric characters, %d-%d characters in length",
				DefaultMinAddrLength,
				DefaultMaxAddrLength,
			)
		}
	}

	return nil
}
