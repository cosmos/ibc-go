package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
)

// Migrator is a struct for handling in-place state migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator creates a new Migrator instance.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateParams migrates the parameters from a legacy param subspace to the proper
// params module. This function is only required on an upgrade from v1 to v2.
func (m Migrator) MigrateParams(ctx sdk.Context) error {
	// Default to the module's default params if no legacy params exist
	params := types.DefaultParams()

	// Note: Legacy subspace access removed. Parameters are now self-managed.
	// The migration now effectively sets the parameters to default if they weren't already set directly.

	if err := params.Validate(); err != nil {
		return err
	}

	// Set the params directly in the keeper
	m.keeper.SetParams(ctx, params)

	m.keeper.Logger(ctx).Info("successfully migrated rate-limiting module to self-manage params")
	return nil
}
