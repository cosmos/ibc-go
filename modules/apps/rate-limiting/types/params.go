package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter keys
var (
	KeyEnabled           = []byte("Enabled")
	KeyDefaultMaxOutflow = []byte("DefaultMaxOutflow")
	KeyDefaultMaxInflow  = []byte("DefaultMaxInflow")
	KeyDefaultPeriod     = []byte("DefaultPeriod") // in seconds
)

// DefaultParams returns default rate-limiting module parameters
func DefaultParams() Params {
	return Params{
		Enabled:           true,
		DefaultMaxOutflow: "1000000", // This is an example value, to be adjusted as needed
		DefaultMaxInflow:  "1000000", // This is an example value, to be adjusted as needed
		DefaultPeriod:     86400,     // Default period of 1 day in seconds
	}
}

// ParamKeyTable returns the parameter key table for the rate-limiting module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns the parameter set pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyEnabled, &p.Enabled, validateBool),
		paramtypes.NewParamSetPair(KeyDefaultMaxOutflow, &p.DefaultMaxOutflow, validateString),
		paramtypes.NewParamSetPair(KeyDefaultMaxInflow, &p.DefaultMaxInflow, validateString),
		paramtypes.NewParamSetPair(KeyDefaultPeriod, &p.DefaultPeriod, validateUint64),
	}
}

// Validate performs basic validation on the parameters
func (p Params) Validate() error {
	if err := validateBool(p.Enabled); err != nil {
		return err
	}
	if err := validateString(p.DefaultMaxOutflow); err != nil {
		return err
	}
	if err := validateString(p.DefaultMaxInflow); err != nil {
		return err
	}
	if err := validateUint64(p.DefaultPeriod); err != nil {
		return fmt.Errorf("default period must be positive: %d", p.DefaultPeriod)
	}
	return nil
}

// String implements fmt.Stringer
func (p Params) String() string {
	return fmt.Sprintf(`Rate Limiting Params:
  Enabled:              %t
  Default Max Outflow:  %s
  Default Max Inflow:   %s
  Default Period:       %d
`, p.Enabled, p.DefaultMaxOutflow, p.DefaultMaxInflow, p.DefaultPeriod)
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateString(i interface{}) error {
	s, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if s == "" {
		return fmt.Errorf("value cannot be empty")
	}
	return nil
}

func validateUint64(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("value cannot be zero")
	}
	return nil
}
