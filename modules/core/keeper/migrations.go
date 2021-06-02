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
// - solo machine clients
// - connections using solo machines
// - channels using solo machines
// - expired tendermint consensus states
//
// Connections are removed if the associated client does not exist.
// Channels are removed if the associated connection does not exist.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	clientMigrator := clientkeeper.NewMigrator(m.keeper.ClientKeeper)
	if err := clientMigrator.Migrate1to2(ctx); err != nil {
		return err
	}

	connectionMigrator := connectionkeeper.NewMigrator(m.keeper.ConnectionKeeper)
	if err := connectionMigrator.Migrate1to2(ctx); err != nil {
		return err
	}

	channelMigrator := channelkeeper.NewMigrator(m.keeper.ChannelKeeper)
	if err := channelMigrator.Migrate1to2(ctx); err != nil {
		return err
	}

	return nil
}
