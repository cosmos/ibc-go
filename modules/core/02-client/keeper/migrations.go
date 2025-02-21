package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/02-client/migrations/v7"
	"github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
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

// Migrate2to3 migrates from consensus version 2 to 3.
// This migration
// - migrates solo machine client states from v2 to v3 protobuf definition
// - prunes solo machine consensus states
// - removes the localhost client
// - asserts that existing tendermint clients are properly registered on the chain codec
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v7.MigrateStore(ctx, m.keeper.storeService, m.keeper.cdc, m.keeper)
}

// MigrateParams migrates from consensus version 4 to 5.
// This migration takes the parameters that are currently stored and managed by x/params
// and stores them directly in the ibc module's state.
func (m Migrator) MigrateParams(ctx sdk.Context) error {
	var params types.Params
	m.keeper.legacySubspace.GetParamSet(ctx, &params)
	if err := params.Validate(); err != nil {
		return err
	}

	m.keeper.SetParams(ctx, params)
	m.keeper.Logger(ctx).Info("successfully migrated client to self-manage params")
	return nil
}

// MigrateToStatelessLocalhost deletes the localhost client state. The localhost
// implementation is now stateless.
func (m Migrator) MigrateToStatelessLocalhost(ctx sdk.Context) error {
	clientStore := m.keeper.ClientStore(ctx, exported.LocalhostClientID)

	// delete the client state
	clientStore.Delete(host.ClientStateKey())
	return nil
}
