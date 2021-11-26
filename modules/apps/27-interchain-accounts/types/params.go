package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	// DefaultControllerEnabled enabled
	DefaultControllerEnabled = true
	// DefaultHostEnabled enabled
	DefaultHostEnabled = true
)

var (
	// KeyControllerEnabled is store's key for ControllerEnabled Params
	KeyControllerEnabled = []byte("ControllerEnabled")
	// KeyHostEnabled is store's key for HostEnabled Params
	KeyHostEnabled = []byte("HostEnabled")
)

// ParamKeyTable type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new parameter configuration for the interchain accounts module
func NewParams(enableController, enableHost bool) Params {
	return Params{
		ControllerEnabled: enableController,
		HostEnabled:       enableHost,
	}
}

// DefaultParams is the default parameter configuration for the ibc-transfer module
func DefaultParams() Params {
	return NewParams(DefaultControllerEnabled, DefaultHostEnabled)
}

// Validate all ibc-transfer module parameters
func (p Params) Validate() error {
	if err := validateEnabled(p.ControllerEnabled); err != nil {
		return err
	}

	return validateEnabled(p.HostEnabled)
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyControllerEnabled, p.ControllerEnabled, validateEnabled),
		paramtypes.NewParamSetPair(KeyHostEnabled, p.HostEnabled, validateEnabled),
	}
}

func validateEnabled(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
