package types

import (
	"fmt"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// TODO: determine sane default value for upgrade timeout.

var DefaultTimeout = NewTimeout(clienttypes.ZeroHeight(), uint64(time.Hour.Nanoseconds()))

// NewParams creates a new parameter configuration for the channel submodule
func NewParams(upgradeTimeout Timeout) Params {
	return Params{
		UpgradeTimeout: upgradeTimeout,
	}
}

// DefaultParams is the default parameter configuration for the channel submodule
func DefaultParams() Params {
	return NewParams(DefaultTimeout)
}

// Validate the params.
func (p Params) Validate() error {
	if !p.UpgradeTimeout.Height.IsZero() {
		return fmt.Errorf("upgrade timeout height must be zero. got : %v", p.UpgradeTimeout.Height)
	}
	if p.UpgradeTimeout.Timestamp == 0 {
		return fmt.Errorf("upgrade timeout timestamp invalid: %v", p.UpgradeTimeout.Timestamp)
	}
	return nil
}
