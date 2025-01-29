package keeper

import (
	"context"

	connectionv7 "github.com/cosmos/ibc-go/v9/modules/core/03-connection/migrations/v7"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate3to4 migrates from version 3 to 4.
// This migration writes the sentinel localhost connection end to state.
func (m Migrator) Migrate3to4(ctx context.Context) error {
	connectionv7.MigrateLocalhostConnection(ctx, m.keeper)
	return nil
}

// MigrateParams migrates from consensus version 4 to 5.
// This migration takes the parameters that are currently stored and managed by x/params
// and stores them directly in the ibc module's state.
func (Migrator) MigrateParams(_ context.Context) error {
	return nil
}
