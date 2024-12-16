package keeper

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper *Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper *Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateParams migrates params to the default channel params.
func (m Migrator) MigrateParams(ctx sdk.Context) error {
	params := channeltypes.DefaultParams()
	m.keeper.SetParams(ctx, params)
	m.keeper.Logger(ctx).Info("successfully migrated ibc channel params")
	return nil
}

// MigrateNextSequenceSend migrates the nextSequenceSend storage from the v1 to v2 format
func (m Migrator) MigrateNextSequenceSend(ctx sdk.Context) error {
	store := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	iterator := storetypes.KVStorePrefixIterator(store, []byte(host.KeyNextSeqSendPrefix))
	m.keeper.IteratePacketSequence(ctx, iterator, func(portID, channelID string, nextSendSeq uint64) bool {
		m.keeper.SetNextSequenceSend(ctx, portID, channelID, nextSendSeq)
		return false
	})
	return nil
}
