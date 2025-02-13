package types

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// UpgradeKeeper expected upgrade keeper
type UpgradeKeeper interface {
	GetUpgradePlan(ctx sdk.Context) (plan upgradetypes.Plan, err error)
	GetUpgradedClient(ctx sdk.Context, height int64) ([]byte, error)
	SetUpgradedClient(ctx sdk.Context, planHeight int64, bz []byte) error
	GetUpgradedConsensusState(ctx sdk.Context, lastHeight int64) ([]byte, error)
	SetUpgradedConsensusState(ctx sdk.Context, planHeight int64, bz []byte) error
	ScheduleUpgrade(ctx sdk.Context, plan upgradetypes.Plan) error
}

// ParamSubspace defines the expected Subspace interface for module parameters.
type ParamSubspace interface {
	GetParamSet(ctx sdk.Context, ps paramtypes.ParamSet)
}
