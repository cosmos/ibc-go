package types

import (
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// TODO: determine sane default value for upgrade timeout.
var DefaultTimeout = NewTimeout(clienttypes.NewHeight(1, 1000), 0)

// NewParams creates a new parameter configuration for the host submodule
func NewParams(upgradeTimeout Timeout) Params {
	return Params{
		UpgradeTimeout: upgradeTimeout,
	}
}

// DefaultParams is the default parameter configuration for the host submodule
func DefaultParams() Params {
	return NewParams(DefaultTimeout)
}

// Validate all ibc-client module parameters
func (p Params) Validate() error {
	if !p.UpgradeTimeout.IsValid() {
		return fmt.Errorf("upgrade timeout invalid: %v", p.UpgradeTimeout)
	}
	return nil
}
