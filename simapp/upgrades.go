package simapp

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/ibc-go/simapp/upgrades"
	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
	packetforwardtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
	ratelimittypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
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

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V11,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V11_1,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V11_1LegacyPFM,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V11_2,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V11_2LegacyIBCApps,
		upgrades.CreateDefaultUpgradeHandler(
			app.ModuleManager,
			app.configurator,
		),
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == upgrades.V11 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{gmptypes.StoreKey},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	// NOTE: Remove gmptypes.StoreKey here if you are upgrading from v11 to v11.1 as v11 upgrade handler will run
	if upgradeInfo.Name == upgrades.V11_1LegacyPFM && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{gmptypes.StoreKey},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	// NOTE: Remove gmptypes.StoreKey here if you are upgrading from v11 to v11.1 as v11 upgrade handler will run
	if upgradeInfo.Name == upgrades.V11_1 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{gmptypes.StoreKey, packetforwardtypes.StoreKey},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	// NOTE: Add gmptypes.StoreKey and packetforwardtypes.StoreKey here if you are upgrading directly from v10
	if upgradeInfo.Name == upgrades.V11_2 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{ratelimittypes.StoreKey},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	// NOTE: The packet forward middleware and rate-limiting store keys are not added here since chains
	// upgrading from the legacy ibc-apps modules already have them under the same store names
	if upgradeInfo.Name == upgrades.V11_2LegacyIBCApps && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{gmptypes.StoreKey},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
