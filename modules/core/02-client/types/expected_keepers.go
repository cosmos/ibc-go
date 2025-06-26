package types

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"
)

// UpgradeKeeper expected upgrade keeper
type UpgradeKeeper interface {
	GetUpgradePlan(ctx context.Context) (plan upgradetypes.Plan, err error)
	GetUpgradedClient(ctx context.Context, height int64) ([]byte, error)
	SetUpgradedClient(ctx context.Context, planHeight int64, bz []byte) error
	GetUpgradedConsensusState(ctx context.Context, lastHeight int64) ([]byte, error)
	SetUpgradedConsensusState(ctx context.Context, planHeight int64, bz []byte) error
	ScheduleUpgrade(ctx context.Context, plan upgradetypes.Plan) error
}
