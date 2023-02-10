package v7

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	consensusparamskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
	ibctmmigrations "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint/migrations"
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
	clientKeeper clientkeeper.Keeper,
	consensusParamsKeeper consensusparamskeeper.Keeper,
	paramsKeeper paramskeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		// OPTIONAL: prune expired tendermint consensus states to save storage space
		if _, err := ibctmmigrations.PruneExpiredConsensusStates(ctx, cdc, clientKeeper); err != nil {
			return nil, err
		}

		legacyBaseAppSubspace, ok := paramsKeeper.GetSubspace(baseapp.Paramspace)
		if !ok {
			return nil, fmt.Errorf("baseapp subspace not found")
		}

		baseapp.MigrateParams(ctx, legacyBaseAppSubspace, &consensusParamsKeeper)

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
