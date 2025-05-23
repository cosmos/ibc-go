package upgrades

import (
	"context"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	consensusparamskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	clientkeeper "github.com/cosmos/ibc-go/v10/modules/core/02-client/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctmmigrations "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint/migrations"
)

const (
	// V7 defines the upgrade name for the ibc-go/v7 upgrade handler.
	V7 = "v7"
	// V7_1 defines the upgrade name for the ibc-go/v7.1 upgrade handler.
	V7_1 = "v7.1"
	// V8 defines the upgrade name for the ibc-go/v8 upgrade handler.
	V8 = "v8"
	// V8_1 defines the upgrade name for the ibc-go/v8.1 upgrade handler.
	V8_1 = "v8.1"
	// V10 defines the upgrade name for the ibc-go/v10 upgrade handler.
	V10 = "v10"
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

// CreateV7UpgradeHandler creates an upgrade handler for the ibc-go/v7 SimApp upgrade.
func CreateV7UpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	cdc codec.BinaryCodec,
	clientKeeper clientkeeper.Keeper,
	consensusParamsKeeper consensusparamskeeper.Keeper,
	paramsKeeper paramskeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(goCtx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		// OPTIONAL: prune expired tendermint consensus states to save storage space
		if _, err := ibctmmigrations.PruneExpiredConsensusStates(ctx, cdc, &clientKeeper); err != nil {
			return nil, err
		}

		legacyBaseAppSubspace := paramsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())
		err := baseapp.MigrateParams(ctx, legacyBaseAppSubspace, consensusParamsKeeper.ParamsStore)
		if err != nil {
			panic(err)
		}

		return mm.RunMigrations(goCtx, configurator, vm)
	}
}

// CreateV7LocalhostUpgradeHandler creates an upgrade handler for the ibc-go/v7.1 SimApp upgrade.
func CreateV7LocalhostUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	clientKeeper clientkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(goCtx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(goCtx)
		// explicitly update the IBC 02-client params, adding the localhost client type
		params := clientKeeper.GetParams(ctx)
		params.AllowedClients = append(params.AllowedClients, exported.Localhost)
		clientKeeper.SetParams(ctx, params)

		return mm.RunMigrations(goCtx, configurator, vm)
	}
}
