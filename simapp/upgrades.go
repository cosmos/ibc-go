package simapp

import (
	"context"

	runtime "github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/ibc-go/simapp/upgrades"
	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
	ratelimittypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
	channelmigrationsv11 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/migrations/v11"
	channeltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
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

	v11Point1UpgradeHandler := func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		ibcStoreService := runtime.NewKVStoreService(app.keys[ibcexported.StoreKey])

		needsChannelMigration := false
		store := ibcStoreService.OpenKVStore(sdkCtx)
		app.IBCKeeper.ChannelKeeper.IterateChannels(sdkCtx, func(ic channeltypes.IdentifiedChannel) bool {
			bz, err := store.Get(channelmigrationsv11.NextSequenceSendV1Key(ic.PortId, ic.ChannelId))
			if err != nil {
				panic(err)
			}
			needsChannelMigration = len(bz) > 0
			return needsChannelMigration
		})

		if needsChannelMigration {
			if err := channelmigrationsv11.MigrateStore(sdkCtx, ibcStoreService, app.appCodec, app.IBCKeeper); err != nil {
				return nil, err
			}
		}

		return app.ModuleManager.RunMigrations(ctx, app.configurator, vm)
	}

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V11_1,
		v11Point1UpgradeHandler,
	)

	app.UpgradeKeeper.SetUpgradeHandler(
		upgrades.V11_1LegacyPFM,
		v11Point1UpgradeHandler,
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

	if upgradeInfo.Name == upgrades.V11_1 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{ratelimittypes.StoreKey},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}

	if upgradeInfo.Name == upgrades.V11_1LegacyPFM && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{gmptypes.StoreKey, ratelimittypes.StoreKey},
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
