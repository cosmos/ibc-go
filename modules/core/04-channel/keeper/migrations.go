package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v10 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/migrations/v10"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m *Migrator) Migrate7To8(ctx sdk.Context) error {
	return v10.MigrateStore(ctx, m.keeper.storeService, m.keeper.cdc, m.keeper)
}
