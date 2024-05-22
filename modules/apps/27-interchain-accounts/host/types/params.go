package types

import (
	"fmt"
	"slices"
	"strings"
)

const (
	// DefaultHostEnabled is the default value for the host param (set to true)
	DefaultHostEnabled = true
	// Maximum length of the allowlist
	MaxAllowListLength = 500
)

// NewParams creates a new parameter configuration for the host submodule
func NewParams(enableHost bool, allowMsgs []string) Params {
	return Params{
		HostEnabled:   enableHost,
		AllowMessages: allowMsgs,
	}
}

// DefaultParams is the default parameter configuration for the host submodule
func DefaultParams() Params {
	return NewParams(DefaultHostEnabled, []string{AllowAllHostMsgs})
}

// Validate validates all host submodule parameters
func (p Params) Validate() error {
	return validateAllowlist(p.AllowMessages)
}

func validateAllowlist(allowMsgs []string) error {
	if len(allowMsgs) > MaxAllowListLength {
		return fmt.Errorf("allow list length must not exceed %d items", MaxAllowListLength)
	}

	if slices.Contains(allowMsgs, AllowAllHostMsgs) && len(allowMsgs) > 1 {
		return fmt.Errorf("allow list must have only one element because the allow all host messages wildcard (%s) is present", AllowAllHostMsgs)
	}

	for _, typeURL := range allowMsgs {
		if strings.TrimSpace(typeURL) == "" {
			return fmt.Errorf("parameter must not contain empty strings: %s", allowMsgs)
		}
	}

	return nil
}
