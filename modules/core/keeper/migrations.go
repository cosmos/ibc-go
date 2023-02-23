package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 migrates from version 2 to 3. See 02-client keeper function Migrate2to3.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	clientKeeper, ok := m.keeper.ClientKeeper.(clientkeeper.Keeper)
	if !ok {
		return fmt.Errorf("failed to assert m.keeper.ClientKeeper to type clientkeeper.Keeper")
	}

	clientMigrator := clientkeeper.NewMigrator(clientKeeper)
	if err := clientMigrator.Migrate2to3(ctx); err != nil {
		return err
	}

	return nil
}
