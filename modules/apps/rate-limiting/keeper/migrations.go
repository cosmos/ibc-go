package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v2 "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/migrations/v2"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates the rate-limiting store from v1 to v2 by:
// - Migrating whitelist entries from the incorrect "address-blacklist" prefix to the correct "address-whitelist" prefix
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.MigrateStore(ctx, m.keeper.storeService, m.keeper.cdc)
}
