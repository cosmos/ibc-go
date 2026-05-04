package types

import (
	"regexp"

	errorsmod "cosmossdk.io/errors"
)

// precompiled for performance
var denomRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// ValidateTokenFactoryDenom validates that a denom is a valid tokenfactory denom
func ValidateTokenFactoryDenom(denom string) error {
	if len(denom) == 0 || len(denom) > 20 {
		return errorsmod.Wrapf(ErrInvalidDenom, "invalid denom, must be between 1 and 20 characters: %s", denom)
	}
	if !denomRegex.MatchString(denom) {
		return errorsmod.Wrapf(ErrInvalidDenom, "invalid denom, must contain only alphanumeric characters: %s", denom)
	}
	return nil
}
