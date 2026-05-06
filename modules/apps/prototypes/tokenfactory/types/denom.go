package types

import (
	"regexp"
	"strings"

	errorsmod "cosmossdk.io/errors"
)

// precompiled for performance
var denomRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// ValidateTokenFactoryDenom validates a subdenom (used as input to MsgCreateDenom).
func ValidateTokenFactoryDenom(denom string) error {
	if len(denom) == 0 || len(denom) > 20 {
		return errorsmod.Wrapf(ErrInvalidDenom, "invalid denom, must be between 1 and 20 characters: %s", denom)
	}
	if !denomRegex.MatchString(denom) {
		return errorsmod.Wrapf(ErrInvalidDenom, "invalid denom, must contain only alphanumeric characters: %s", denom)
	}
	return nil
}

// ValidateFullTokenFactoryDenom validates a full tokenfactory denom of the form
// factory/<creator>/<subdenom>.
func ValidateFullTokenFactoryDenom(denom string) error {
	parts := strings.SplitN(denom, "/", 3)
	if len(parts) != 3 || parts[0] != "factory" || parts[1] == "" {
		return errorsmod.Wrapf(ErrInvalidDenom, "invalid tokenfactory denom, expected factory/<creator>/<subdenom>, got: %s", denom)
	}
	return ValidateTokenFactoryDenom(parts[2])
}
