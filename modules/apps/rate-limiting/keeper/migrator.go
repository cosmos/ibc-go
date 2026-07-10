package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/migrations/v2"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator creates a new Migrator instance.
func NewMigrator(k *Keeper) Migrator {
	return Migrator{keeper: k}
}

// Migrate1to2 clears legacy pending packet markers so new denom-scoped
// collections pending packet state starts empty. Flow charged by pre-upgrade
// pending packets is not compensated and remains charged until the next quota
// reset because the legacy markers do not include denoms.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.Migrate(ctx, m.keeper.storeService)
}
