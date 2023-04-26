package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/exported"
	v8 "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/migrations/v8"
)

// Migrator is a struct for handling in-place state migrations.
type Migrator struct {
	keeper         *Keeper
	legacySubspace exported.Subspace
}

// NewMigrator returns Migrator instance for the state migration.
func NewMigrator(k *Keeper, ss exported.Subspace) Migrator {
	return Migrator{
		keeper:         k,
		legacySubspace: ss,
	}
}

// Migrate2to3 migrates the 27-interchain-accounts module state from the
// consensus version 2 to version 3. Specifically, it takes the parameters that
// are currently stored and managed by the x/params modules and stores them directly
// into the host submodule state.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v8.Migrate(ctx, ctx.KVStore(m.keeper.storeKey), m.legacySubspace, m.keeper.cdc)
}
