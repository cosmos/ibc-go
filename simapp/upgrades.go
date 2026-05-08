package simapp

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/ibc-go/simapp/upgrades"
	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
	packetforwardtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
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

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		var storeUpgrades storetypes.StoreUpgrades

		switch upgradeInfo.Name {
		case upgrades.V11:
			storeUpgrades.Added = []string{gmptypes.StoreKey}
		case upgrades.V11_1:
			storeUpgrades.Added = []string{packetforwardtypes.StoreKey}
		default:
			return
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
