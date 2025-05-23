package simapp

import (
	storetypes "cosmossdk.io/store/types"
	circuittypes "github.com/cosmos/cosmos-sdk/x/circuit/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"

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

	if upgradeInfo.Name == upgrades.V7 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				consensusparamtypes.StoreKey,
				crisistypes.StoreKey,
			},
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
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
