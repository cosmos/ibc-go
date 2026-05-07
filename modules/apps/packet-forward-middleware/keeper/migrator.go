package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/migrations/v3"
	"github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/migrations/v4"
)

// Migrator is a struct for handling in-place state migrations.
type Migrator struct {
	keeper *Keeper
}

func NewMigrator(k *Keeper) Migrator {
	return Migrator{
		keeper: k,
	}
}

// Migrate2to3 migrates the module state from the consensus version 2 to
// version 3
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v3.Migrate(ctx, m.keeper.bankKeeper, m.keeper.channelKeeper, m.keeper.transferKeeper)
}

// Migrate3to4 migrates the module state from the consensus version 3 to
// version 4.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	return v4.Migrate(ctx, m.keeper.storeService, m.keeper.cdc)
}
