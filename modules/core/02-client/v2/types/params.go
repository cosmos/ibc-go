package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Maximum length of the allowed relayers list
const MaxAllowedRelayersLength = 20

// NewParams instantiates a new allowed relayer list for a client with provided addresses
func NewParams(allowedRelayers ...string) Params {
	return Params{
		AllowedRelayers: allowedRelayers,
	}
}

// DefaultParams is empty and therefore permissionless
func DefaultParams() Params {
	return NewParams()
}

// Validate ensures all provided addresses are valid sdk Addresses
func (p Params) Validate() error {
	return validateRelayers(p.AllowedRelayers)
}

// IsAllowedRelayer checks if the given address is registered on the allowlist.
func (p Params) IsAllowedRelayer(relayer sdk.AccAddress) bool {
	if len(p.AllowedRelayers) == 0 {
		return true
	}
	for _, r := range p.AllowedRelayers {
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
