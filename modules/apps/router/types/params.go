package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	DefaultFeePercentage = sdk.NewDec(0)
	// KeyFeePercentage is store's key for FeePercentage Params
	KeyFeePercentage = []byte("FeePercentage")
)

// ParamKeyTable type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new parameter configuration for the ibc transfer module
func NewParams(feePercentage sdk.Dec) Params {
	return Params{
		FeePercentage: feePercentage,
	}
}

// DefaultParams is the default parameter configuration for the ibc-transfer module
func DefaultParams() Params {
	return NewParams(DefaultFeePercentage)
}

// Validate all ibc-transfer module parameters
func (p Params) Validate() error {
	return validateFeePercentage(p.FeePercentage)
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyFeePercentage, p.FeePercentage, validateFeePercentage),
	}
}

func validateFeePercentage(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if !(v.IsPositive() && v.LT(sdk.OneDec())) {
		return fmt.Errorf("invalid range for fee percentage. expected between 0 and 1 got %d", v.RoundInt64())
	}

	return nil
}
