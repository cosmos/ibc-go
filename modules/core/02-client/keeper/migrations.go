package keeper

import (
	"context"

	"github.com/cosmos/ibc-go/v9/modules/core/02-client/migrations/v7"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 migrates from consensus version 2 to 3.
// This migration
// - migrates solo machine client states from v2 to v3 protobuf definition
// - prunes solo machine consensus states
// - removes the localhost client
// - asserts that existing tendermint clients are properly registered on the chain codec
func (m Migrator) Migrate2to3(ctx context.Context) error {
	return v7.MigrateStore(ctx, m.keeper.Logger, m.keeper.KVStoreService, m.keeper.cdc, m.keeper)
}

// MigrateParams migrates from consensus version 4 to 5.
// This migration takes the parameters that are currently stored and managed by x/params
// and stores them directly in the ibc module's state.
func (Migrator) MigrateParams(_ context.Context) error {
	return nil
}

// MigrateToStatelessLocalhost deletes the localhost client state. The localhost
// implementation is now stateless.
func (m Migrator) MigrateToStatelessLocalhost(ctx context.Context) error {
	clientStore := m.keeper.ClientStore(ctx, exported.LocalhostClientID)

	// delete the client state
	clientStore.Delete(host.ClientStateKey())
	return nil
}
