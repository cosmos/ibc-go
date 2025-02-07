package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	controllertypes "github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/controller/types"
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
func (m Migrator) MigrateParams(ctx context.Context) error {
	if m.keeper != nil {
		params := controllertypes.DefaultParams()
		if m.keeper.legacySubspace != nil {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			m.keeper.legacySubspace.GetParamSetIfExists(sdkCtx, &params)
		}
		m.keeper.SetParams(ctx, params)
		m.keeper.Logger(ctx).Info("successfully migrated ica/controller submodule to self-manage params")
	}
	return nil
}
