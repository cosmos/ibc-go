package keeper

import (
	"context"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns Migrator instance for the state migration.
func NewMigrator(k *Keeper) Migrator {
	return Migrator{
		keeper: k,
	}
}

// MigrateParams migrates the controller submodule's parameters from the x/params to self store.
func (Migrator) MigrateParams(_ context.Context) error {
	return nil
}
