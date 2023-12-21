package types

import (
	"fmt"
	"time"

	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

// DefaultTimeout defines a default parameter for the channel upgrade protocol.
// It allows relayers a window in which they can flush all in-flight packets on a channel before completing the upgrade handshake.
// This parameter can be overridden by a valid authority using the UpdateChannelParams rpc.
var DefaultTimeout = NewTimeout(clienttypes.ZeroHeight(), uint64(10*time.Minute.Nanoseconds()))

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
