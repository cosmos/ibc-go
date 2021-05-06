package types

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// DefaultTimePerBlock is the default value for maximum time
// expected per block.
const DefaultTimePerBlock = 10 * time.Minute

// KeyMaxTimePerBlock is store's key for MaxTimePerBlock parameter
var KeyMaxTimePerBlock = []byte("MaxTimePerBlock")

// ParamKeyTable type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new parameter configuration for the ibc connection module
func NewParams(timePerBlock uint64) Params {
	return Params{
		MaxTimePerBlock: timePerBlock,
	}
}

// DefaultParams is the default parameter configuration for the ibc connection module
func DefaultParams() Params {
	return NewParams(uint64(DefaultTimePerBlock))
}

// Validate is a no-op for connection parameters
func (p Params) Validate() error {
	return nil
}

// ParamSetPairs implements params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxTimePerBlock, p.MaxTimePerBlock, validateParams),
	}
}

func validateParams(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter. expected %T, got type: %T", uint64(1), i)
	}
	return nil
}
