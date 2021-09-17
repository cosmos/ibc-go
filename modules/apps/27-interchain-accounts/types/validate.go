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

// ValidateVersion performs basic validation of the provided ics27 version string
// When no delimiter is present it compares against the expected version
func ValidateVersion(version string) error {
	s := strings.Split(version, Delimiter)

	if s[0] != VersionPrefix {
		return sdkerrors.Wrapf(ErrInvalidVersion, "expected %s, got %s", VersionPrefix, s[0])
	}

	if len(s) > 1 {
		if len(s) != 2 {
			return sdkerrors.Wrap(ErrInvalidAccountAddress, "unexpected address format")
		}

		if !IsValidAddr(s[1]) || len(s[1]) == 0 || len(s[1]) > DefaultMaxAddrLength {
			return sdkerrors.Wrapf(
				ErrInvalidAccountAddress,
				"address must contain strictly alphanumeric characters, not exceeding %d characters in length",
				DefaultMaxAddrLength,
			)
		}
	}

	return nil
}
