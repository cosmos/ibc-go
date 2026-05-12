package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	v2 "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/migrations/v2"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

func NewMigrator(k *Keeper) Migrator {
	return Migrator{keeper: k}
}

// Migrate1to2 widens the PendingSendPacket key's channel-ID segment from
// 16 to 64 bytes so IBC v2 channel IDs fit.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.Migrate(ctx, m.keeper.storeService)
}
