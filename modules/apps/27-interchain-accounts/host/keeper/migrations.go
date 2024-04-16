package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
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
func (m Migrator) MigrateParams(ctx sdk.Context) error {
	if m.keeper != nil {
		defaultParams := types.DefaultParams()
		var params types.Params
		ps := &params
		for _, pair := range ps.ParamSetPairs() {
			if !m.keeper.legacySubspace.Has(ctx, pair.Key) {
				var value interface{}
				if string(pair.Key) == "HostEnabled" {
					value = defaultParams.HostEnabled
				} else if string(pair.Key) == "AllowMessages" {
					value = defaultParams.AllowMessages
				} else {
					continue
				}
				m.keeper.legacySubspace.Set(ctx, pair.Key, value)
			}
			m.keeper.legacySubspace.Get(ctx, pair.Key, pair.Value)
		}
		if err := params.Validate(); err != nil {
			return err
		}
		m.keeper.SetParams(ctx, params)
		m.keeper.Logger(ctx).Info("successfully migrated ica/host submodule to self-manage params")
	}
	return nil
}
