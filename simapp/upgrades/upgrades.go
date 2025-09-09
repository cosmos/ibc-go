package upgrades

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	v11 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v11"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
)

const (
	// V8 defines the upgrade name for the ibc-go/v8 upgrade handler.
	V8 = "v8"
	// V8_1 defines the upgrade name for the ibc-go/v8.1 upgrade handler.
	V8_1 = "v8.1"
	// V10 defines the upgrade name for the ibc-go/v10 upgrade handler.
	V10 = "v10"
	// V11 defines the upgrade name for the ibc-go/v11 upgrade handler.
	V11 = "v11"
)

// CreateDefaultUpgradeHandler creates an upgrade handler which can be used for regular upgrade tests
// that do not require special logic
func CreateDefaultUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, vm)
	}
}

// CreateV11UpgradeHandler creates an upgrade handler for v11 that includes IBC sequence migration
func CreateV11UpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	app interface{},
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		// Run module migrations
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, fmt.Errorf("failed to run module migrations: %w", err)
		}

		// Perform IBC v11 migration
		if err := performIBCV11Migration(ctx, app); err != nil {
			return vm, fmt.Errorf("failed to perform IBC v11 migration: %w", err)
		}

		return vm, nil
	}
}

// performIBCV11Migration performs the IBC v11 sequence migration
func performIBCV11Migration(ctx context.Context, app interface{}) error {
	// Type assertion to get the app methods we need
	simApp, ok := app.(interface {
		GetIBCKeeper() *ibckeeper.Keeper
		AppCodec() codec.Codec
		GetKey(moduleName string) *storetypes.KVStoreKey
	})
	if !ok {
		return fmt.Errorf("invalid app type for IBC migration")
	}

	// Get the store service
	storeService := runtime.NewKVStoreService(simApp.GetKey(ibcexported.StoreKey))

	// Try to perform the migration
	err := v11.MigrateStore(
		sdk.UnwrapSDKContext(ctx),
		storeService,
		simApp.AppCodec(),
		simApp.GetIBCKeeper(),
	)

	return err
}
