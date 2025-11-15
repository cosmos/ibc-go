package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateToStatelessLocalhost deletes the localhost client state. The localhost
// implementation is now stateless.
func (m Migrator) MigrateToStatelessLocalhost(ctx sdk.Context) error {
	clientStore := m.keeper.ClientStore(ctx, exported.LocalhostClientID)

	// delete the client state
	clientStore.Delete(host.ClientStateKey())
	return nil
}
