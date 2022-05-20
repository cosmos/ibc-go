package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// StakingKeeper expected staking keeper
type StakingKeeper interface {
	GetHistoricalInfo(ctx sdk.Context, height int64) (stakingtypes.HistoricalInfo, bool)
	UnbondingTime(ctx sdk.Context) time.Duration
	GetValidators(ctx sdk.Context, maxRetrieve uint32) (validators []stakingtypes.Validator)
	Delegate(
		ctx sdk.Context, delAddr sdk.AccAddress, bondAmt sdk.Int, tokenSrc stakingtypes.BondStatus,
		validator stakingtypes.Validator, subtractAccount bool,
	) (newShares sdk.Dec, err error)
}

// UpgradeKeeper expected upgrade keeper
type UpgradeKeeper interface {
	ClearIBCState(ctx sdk.Context, lastHeight int64)
	GetUpgradePlan(ctx sdk.Context) (plan upgradetypes.Plan, havePlan bool)
	GetUpgradedClient(ctx sdk.Context, height int64) ([]byte, bool)
	SetUpgradedClient(ctx sdk.Context, planHeight int64, bz []byte) error
	GetUpgradedConsensusState(ctx sdk.Context, lastHeight int64) ([]byte, bool)
	SetUpgradedConsensusState(ctx sdk.Context, planHeight int64, bz []byte) error
	ScheduleUpgrade(ctx sdk.Context, plan upgradetypes.Plan) error
}
