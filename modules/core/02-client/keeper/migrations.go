package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v100 "github.com/cosmos/ibc-go/v6/modules/core/02-client/legacy/v100"
	"github.com/cosmos/ibc-go/v6/modules/core/02-client/migrations/v7"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to 2.
// This migration
// - migrates solo machine client states from v1 to v2 protobuf definition
// - prunes solo machine consensus states
// - prunes expired tendermint consensus states
// - adds iteration and processed height keys for unexpired tendermint consensus states
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v100.MigrateStore(ctx, m.keeper.storeKey, m.keeper.cdc)
}

// Migrate2to3 migrates from version 2 to 3.
// This migration
// - migrates solo machine client states from v2 to v3 protobuf definition
// - prunes solo machine consensus states
// - removes the localhost client
// - asserts that existing tendermint clients are properly registered on the chain codec
// - adds iteration and processed height keys for unexpired tendermint consensus states
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v7.MigrateStore(ctx, m.keeper.storeKey, m.keeper.cdc)
}
