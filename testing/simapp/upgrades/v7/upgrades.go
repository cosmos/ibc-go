package v7

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	v7 "github.com/cosmos/ibc-go/v6/modules/core/02-client/migrations/v7"
	ibctm "github.com/cosmos/ibc-go/v6/modules/light-clients/07-tendermint"
)

const (
	// UpgradeName defines the on-chain upgrade name for the SimApp v7 upgrade.
	UpgradeName = "v7"
)

// CreateUpgradeHandler creates an upgrade handler for the v7 SimApp upgrade.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	cdc codec.BinaryCodec,
	hostStoreKey *storetypes.KVStoreKey,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		// Perform in-place store migrations for the v7 upgrade
		// The migration includes:
		//
		// - Migrating solo machine client states from v2 to v3 protobuf definition
		// - Pruning all solo machine consensus states
		// - Removing the localhost client
		// - Asserting existing tendermint clients are properly registered on the chain codec
		if err := v7.MigrateStore(ctx, hostStoreKey, cdc); err != nil {
			return nil, err
		}

		// OPTIONAL: prune expired tendermint consensus states to save storage space
		if err := ibctm.PruneTendermintConsensusStates(ctx, cdc, hostStoreKey); err != nil {
			return nil, err
		}
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
