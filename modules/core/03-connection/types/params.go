package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// DefaultTimePerBlock is the default value for expected time per block.
const DefaultTimePerBlock = 30 * time.Second

// KeyExpectedTimePerBlock is store's key for ExpectedTimePerBlock parameter
var KeyExpectedTimePerBlock = []byte("ExpectedTimePerBlock")

// ParamKeyTable type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new parameter configuration for the ibc connection module
func NewParams(timePerBlock uint64) Params {
	return Params{
		ExpectedTimePerBlock: timePerBlock,
	}
}

// DefaultParams is the default parameter configuration for the ibc connection module
func DefaultParams() Params {
	return NewParams(uint64(DefaultTimePerBlock))
}

// Validate ensures ExpectedTimePerBlock is non-zero
func (p Params) Validate() error {
	if p.ExpectedTimePerBlock == 0 {
		return fmt.Errorf("ExpectedTimePerBlock cannot be zero")
	}
	return nil
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyExpectedTimePerBlock, p.ExpectedTimePerBlock, validateParams),
	}
}

func validateParams(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter. expected %T, got type: %T", uint64(1), i)
	}
	return nil
}
