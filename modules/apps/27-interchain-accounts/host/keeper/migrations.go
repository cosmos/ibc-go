package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
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

// SetLegacySubspace sets the legacy parameter subspace for the Migrator.
func (m Migrator) SetLegacySubspace(legacySubspace icatypes.ParamSubspace) {
	m.keeper.legacySubspace = legacySubspace
}

// MigrateParams migrates the host submodule's parameters from the x/params to self store.
func (m Migrator) MigrateParams(ctx sdk.Context) error {
	if m.keeper != nil {
		var params types.Params
		if m.keeper.legacySubspace == nil {
			params = types.DefaultParams()
		} else {
			m.keeper.legacySubspace.GetParamSet(ctx, &params)
			if err := params.Validate(); err != nil {
				return err
			}
		}
		m.keeper.SetParams(ctx, params)
		m.keeper.Logger(ctx).Info("successfully migrated ica/host submodule to self-manage params")
	}
	return nil
}
