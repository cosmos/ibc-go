package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateParams migrates params to the specified channel params.
func (m Migrator) MigrateParams(ctx sdk.Context, params channeltypes.Params) error {
	m.keeper.SetParams(ctx, params)
	if err := params.Validate(); err != nil {
		return err
	}
	m.keeper.Logger(ctx).Info("successfully migrated channel params")
	return nil
}

// MigrateDefaultParams migrates params to the default channel params.
func (m Migrator) MigrateDefaultParams(ctx sdk.Context) error {
	return m.MigrateParams(ctx, channeltypes.DefaultParams())
}
