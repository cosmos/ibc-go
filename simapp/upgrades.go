package simapp

import (
	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/x/accounts"
	consensusparamtypes "cosmossdk.io/x/consensus/types"
	pooltypes "cosmossdk.io/x/protocolpool/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/ibc-go/simapp/upgrades"
)

// registerUpgradeHandlers registers all supported upgrade handlers
func (app *SimApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V7,
		upgrades.CreateV7UpgradeHandler(
			app.ModuleManager,
			app.configurator,
			app.appCodec,
			*app.IBCKeeper.ClientKeeper,
			app.ConsensusParamsKeeper,
			app.ParamsKeeper,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V7_1,
		upgrades.CreateV7LocalhostUpgradeHandler(app.ModuleManager, app.configurator, *app.IBCKeeper.ClientKeeper),
	)

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
		upgrades.V9,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == upgrades.V7 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := corestore.StoreUpgrades{
			Added: []string{
				consensusparamtypes.StoreKey,
			},
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	if upgradeInfo.Name == upgrades.V8 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := corestore.StoreUpgrades{}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	if upgradeInfo.Name == upgrades.V10 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := corestore.StoreUpgrades{
			Added: []string{
				pooltypes.StoreKey,
				accounts.StoreKey,
			},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
