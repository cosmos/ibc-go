package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	// KeyDistributionAddress is the store key for the DistributionAddress param
	KeyDistributionAddress = []byte("DistributionAddress")
)

// ParamKeyTable type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new parameter configuration for the host submodule
func NewParams(distAddr string) Params {
	return Params{
		DistributionAddress: distAddr,
	}
}

// DefaultParams is the default parameter configuration for the host submodule
func DefaultParams() Params {
	return Params{}
}

// Validate validates all host submodule parameters
func (p Params) Validate() error {
	if err := validateDistributionAddr(p.DistributionAddress); err != nil {
		return err
	}

	return nil
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyDistributionAddress, p.DistributionAddress, validateDistributionAddr),
	}
}

func validateDistributionAddr(i interface{}) error {
	_, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
