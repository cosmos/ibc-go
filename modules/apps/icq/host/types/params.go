package types

import (
	"fmt"
	"strings"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	// DefaultHostEnabled is the default value for the host param (set to true)
	DefaultHostEnabled = true
	// DefaultAllowHeight is the default value for the allow height param (set to false)
	DefaultAllowHeight = false
	// DefaultAllowProof is the default value for the allow proof param (set to false)
	DefaultAllowProof = false
)

var (
	// KeyHostEnabled is the store key for HostEnabled Params
	KeyHostEnabled = []byte("HostEnabled")
	// KeyAllowQueries is the store key for the AllowQueries Params
	KeyAllowQueries = []byte("AllowQueries")
	// KeyAllowHeight is the store key for the AllowHeight Params
	KeyAllowHeight = []byte("AllowHeight")
	// KeyAllowProof is the store key for the AllowProof Params
	KeyAllowProof = []byte("AllowProof")
)

// ParamKeyTable type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new parameter configuration for the host submodule
func NewParams(enableHost, allowHeight, allowProof bool, allowQueries []string) Params {
	return Params{
		HostEnabled:  enableHost,
		AllowHeight:  allowHeight,
		AllowProof:   allowProof,
		AllowQueries: allowQueries,
	}
}

// DefaultParams is the default parameter configuration for the host submodule
func DefaultParams() Params {
	return NewParams(DefaultHostEnabled, DefaultAllowHeight, DefaultAllowProof, nil)
}

// Validate validates all host submodule parameters
func (p Params) Validate() error {
	if err := validateEnabled(p.HostEnabled); err != nil {
		return err
	}

	if err := validateEnabled(p.AllowHeight); err != nil {
		return err
	}

	if err := validateEnabled(p.AllowProof); err != nil {
		return err
	}

	if err := validateAllowlist(p.AllowQueries); err != nil {
		return err
	}

	return nil
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyHostEnabled, p.HostEnabled, validateEnabled),
		paramtypes.NewParamSetPair(KeyAllowHeight, p.AllowHeight, validateEnabled),
		paramtypes.NewParamSetPair(KeyAllowProof, p.AllowProof, validateEnabled),
		paramtypes.NewParamSetPair(KeyAllowQueries, p.AllowQueries, validateAllowlist),
	}
}

func validateEnabled(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateAllowlist(i interface{}) error {
	allowQueries, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, path := range allowQueries {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("parameter must not contain empty strings: %s", allowQueries)
		}
	}

	return nil
}
