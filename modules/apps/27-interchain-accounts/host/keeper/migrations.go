package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v9/modules/apps/27-interchain-accounts/host/types"
)

// Migrator is a struct for handling in-place state migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns Migrator instance for the state migration.
func NewMigrator(k *Keeper) Migrator {
	return Migrator{
		keeper: k,
	}
}

// MigrateParams migrates the host submodule's parameters from the x/params to self store.
func (m Migrator) MigrateParams(ctx context.Context) error {
	if m.keeper != nil {
		params := types.DefaultParams()
		if m.keeper.legacySubspace != nil {
			// NOTE: legacy params still rely on sdk context
			m.keeper.legacySubspace.GetParamSetIfExists(sdk.UnwrapSDKContext(ctx), &params)
		}
		if err := params.Validate(); err != nil {
			return err
		}
		m.keeper.SetParams(ctx, params)
		m.keeper.Logger.Info("successfully migrated ica/host submodule to self-manage params")
	}
	return nil
}
