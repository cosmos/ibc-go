package keeper

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateParams migrates params to the default channel params.
func (Migrator) MigrateParams(ctx sdk.Context) error {
	return errorsmod.Wrap(ibcerrors.ErrInvalidVersion, "must migrate to ibc-go v8.x first")
}
