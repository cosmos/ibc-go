package types

import (
	"fmt"
)

const (
	// DefaultControllerEnabled is the default value for the controller param (set to true)
	DefaultControllerEnabled = true
)

// KeyControllerEnabled is the store key for ControllerEnabled Params
var KeyControllerEnabled = []byte("ControllerEnabled")

// NewParams creates a new parameter configuration for the controller submodule
func NewParams(enableController bool) Params {
	return Params{
		ControllerEnabled: enableController,
	}
}

// DefaultParams is the default parameter configuration for the controller submodule
func DefaultParams() Params {
	return NewParams(DefaultControllerEnabled)
}

// Validate validates all controller submodule parameters
func (p Params) Validate() error {
	return validateEnabledType(p.ControllerEnabled)
}

func validateEnabledType(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
