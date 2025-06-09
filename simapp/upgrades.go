package simapp

import (
	storetypes "cosmossdk.io/store/types"
	circuittypes "cosmossdk.io/x/circuit/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/ibc-go/simapp/upgrades"
)

// registerUpgradeHandlers registers all supported upgrade handlers
func (app *SimApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V8,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V8_1,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V10,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == upgrades.V8 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				circuittypes.ModuleName,
			},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
