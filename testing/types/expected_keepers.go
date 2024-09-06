package types

import (
	"context"

	stakingtypes "cosmossdk.io/x/staking/types"
)

// StakingKeeper defines the expected staking keeper interface used in the
// IBC testing package
type StakingKeeper interface {
	GetHistoricalInfo(ctx context.Context, height int64) (stakingtypes.HistoricalInfo, error)
}
