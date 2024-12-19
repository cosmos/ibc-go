package types

import (
	"context"

	paramtypes "cosmossdk.io/x/params/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	GetParamSet(ctx sdk.Context, ps paramtypes.ParamSet)
}
