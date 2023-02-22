package upgrades

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const (
	// DefaultUpgradeName is the default upgrade name used for upgrade tests which do not require special handling.
	DefaultUpgradeName = "normal upgrade"
	ClientUpgradeName  = "e2e-client-upgrade"
)

// CreateDefaultUpgradeHandler creates an upgrade handler which can be used for regular upgrade tests
// that do not require special logic
func CreateDefaultUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, vm)
	}
}

// CreateClientUpgradeHandler create an upgrade handler which is supposed to be used in an e2e test which makes
// chain specific changes (in this case the unbonding period) in order to carry out a client upgrade.
func CreateClientUpgradeHandler(mm *module.Manager, configurator module.Configurator, stakingKeeper *stakingkeeper.Keeper) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		params := stakingtypes.DefaultParams()
		params.UnbondingTime = time.Hour * 24 * 7 * 4
		if err := stakingKeeper.SetParams(ctx, params); err != nil {
			return module.VersionMap{}, err
		}
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
