package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Maximum length of the allowed relayers list
const MaxAllowedRelayersLength = 20

// NewConfig instantiates a new allowed relayer list for a client with provided addresses
func NewConfig(allowedRelayers ...string) Config {
	return Config{
		AllowedRelayers: allowedRelayers,
	}
}

// DefaultConfig is empty and therefore permissionless
func DefaultConfig() Config {
	return NewConfig()
}

// Validate ensures all provided addresses are valid sdk Addresses
func (c Config) Validate() error {
	return validateRelayers(c.AllowedRelayers)
}

// IsAllowedRelayer checks if the given address is registered on the allowlist.
func (c Config) IsAllowedRelayer(relayer sdk.AccAddress) bool {
	if len(c.AllowedRelayers) == 0 {
		return true
	}
	for _, r := range c.AllowedRelayers {
		if relayer.Equals(sdk.MustAccAddressFromBech32(r)) {
			return true
		}
	}
	return false
}

func validateRelayers(allowedRelayers []string) error {
	if len(allowedRelayers) > MaxAllowedRelayersLength {
		return fmt.Errorf("allowed relayers length must not exceed %d items", MaxAllowedRelayersLength)
	}

	for _, r := range allowedRelayers {
		if _, err := sdk.AccAddressFromBech32(r); err != nil {
			return fmt.Errorf("invalid relayer address: %s", r)
		}
	}
	return nil
}
