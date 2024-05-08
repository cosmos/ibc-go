package types

import (
	"fmt"
	"strings"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	// DefaultHostEnabled is the default value for the host param (set to true)
	DefaultHostEnabled = true
	// Maximum length of the allowlist
	MaxAllowListLength = 500
)

var (
	// KeyHostEnabled is the store key for HostEnabled Params
	KeyHostEnabled = []byte("HostEnabled")
	// KeyAllowMessages is the store key for the AllowMessages Params
	KeyAllowMessages = []byte("AllowMessages")
)

// ParamKeyTable type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

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
	if err := validateEnabledType(p.HostEnabled); err != nil {
		return err
	}

	if err := validateAllowlist(p.AllowMessages); err != nil {
		return err
	}

	return nil
}

<<<<<<< HEAD
// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyHostEnabled, p.HostEnabled, validateEnabledType),
		paramtypes.NewParamSetPair(KeyAllowMessages, p.AllowMessages, validateAllowlist),
	}
}

func validateEnabledType(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateAllowlist(i interface{}) error {
	allowMsgs, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
=======
func validateAllowlist(allowMsgs []string) error {
	if len(allowMsgs) > MaxAllowListLength {
		return fmt.Errorf("allow list length must not exceed %d items", MaxAllowListLength)
	}

	if slices.Contains(allowMsgs, AllowAllHostMsgs) && len(allowMsgs) > 1 {
		return fmt.Errorf("allow list must have only one element because the allow all host messages wildcard (%s) is present", AllowAllHostMsgs)
>>>>>>> 478f4c60 (imp: check length of slices of messages (#6256))
	}

	for _, typeURL := range allowMsgs {
		if strings.TrimSpace(typeURL) == "" {
			return fmt.Errorf("parameter must not contain empty strings: %s", allowMsgs)
		}
	}

	return nil
}
