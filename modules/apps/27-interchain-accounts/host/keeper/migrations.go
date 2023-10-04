package keeper

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
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
		legacySubpsace, ok := m.keeper.legacySubspace.(paramtypes.Subspace)
		if !ok {
			return errorsmod.Wrapf(ibcerrors.ErrInvalidType, "expected %T, got %T", paramtypes.Subspace{}, m.keeper.legacySubspace)
		}

		var params types.Params
		legacySubpsace.GetParamSet(ctx, &params)

		if err := params.Validate(); err != nil {
			return err
		}
		m.keeper.SetParams(ctx, params)
		m.keeper.Logger(ctx).Info("successfully migrated ica/host submodule to self-manage params")
	}
	return nil
}
