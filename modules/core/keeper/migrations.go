package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	clientkeeper "github.com/cosmos/ibc-go/modules/core/02-client/keeper"
	connectionkeeper "github.com/cosmos/ibc-go/modules/core/03-connection/keeper"
	channelkeeper "github.com/cosmos/ibc-go/modules/core/04-channel/keeper"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to 2.
// This migration prunes:
// - solo machine consensus states
// - expired tendermint consensus states
//
// This migration migrates:
// - solo machine client state from protobuf definition v1 to v2
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	clientMigrator := clientkeeper.NewMigrator(m.keeper.ClientKeeper)
	if err := clientMigrator.Migrate1to2(ctx); err != nil {
		return err
	}

	return nil
}
