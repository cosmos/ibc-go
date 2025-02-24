package v10

import (
	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
)

const (
	// ParamsKey defines the key to store the params in the keeper.
	ParamsKey               = "channelParams"
	KeyPruningSequenceStart = "pruningSequenceStart"

	KeyChannelUpgradePrefix = "channelUpgrades"
)

func MigrateStore(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec, channelKeeper ChannelKeeper) error {
	store := storeService.OpenKVStore(ctx)

	if err := handleChannelMigration(ctx, store, cdc, channelKeeper); err != nil {
		return err
	}
	if err := deleteChannelUpgrades(store); err != nil {
		return err
	}
	if err := deleteParams(store); err != nil {
		return err
	}
	if err := deletePruneSequences(store); err != nil {
		return err
	}

	// TODO: See if there is more to migrate/delete from store

	return nil
}

func handleChannelMigration(ctx sdk.Context, store corestore.KVStore, cdc codec.BinaryCodec, channelKeeper ChannelKeeper) error {
	// Remove channel upgrade sequences and set in-upgrade channels back to open
	iterator := storetypes.KVStorePrefixIterator(runtime.KVStoreAdapter(store), []byte(host.KeyChannelEndPrefix))

	defer sdk.LogDeferred(channelKeeper.Logger(ctx), func() error { return iterator.Close() })
	for ; iterator.Valid(); iterator.Next() {
		var channel Channel
		cdc.MustUnmarshal(iterator.Value(), &channel)

		if channel.State == FLUSHING || channel.State == FLUSHCOMPLETE {
			channel.State = OPEN
		}

		newChannel := types.Channel{
			State:    types.State(channel.State),
			Ordering: types.Order(channel.Ordering),
			Counterparty: types.Counterparty{
				PortId:    channel.Counterparty.PortId,
				ChannelId: channel.Counterparty.ChannelId,
			},
			ConnectionHops: channel.ConnectionHops,
			Version:        channel.Version,
		}
		// Any pitfalls of doing this?
		if err := newChannel.ValidateBasic(); err != nil {
			return err
		}
		portID, channelID := host.MustParseChannelPath(string(iterator.Key()))
		channelKeeper.SetChannel(ctx, portID, channelID, newChannel)
	}

	return nil
}

func deleteChannelUpgrades(store corestore.KVStore) error {
	// Delete channel upgrades (i.e. upgrades, counterparty upgrades, upgrade errors, which are stored in the channelUpgrades prefix)
	return store.Delete([]byte(KeyChannelUpgradePrefix))
}

func deleteParams(store corestore.KVStore) error {
	// Delete channel params
	return store.Delete([]byte(ParamsKey))
}

func deletePruneSequences(store corestore.KVStore) error {
	// Delete all pruning sequences
	return store.Delete([]byte(KeyPruningSequenceStart))
}
