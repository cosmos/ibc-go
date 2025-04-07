package v10

import (
	"errors"
	fmt "fmt"

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
	KeyUpgradePrefix        = "upgrades"
	KeyUpgradeErrorPrefix   = "upgradeError"
	KeyCounterpartyUpgrade  = "counterpartyUpgrade"
)

// PruningSequenceStartKey returns the store key for the pruning sequence start of a particular channel
func PruningSequenceStartKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s", KeyPruningSequenceStart, channelPath(portID, channelID))
}

func ChannelUpgradeKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s/%s", KeyChannelUpgradePrefix, KeyUpgradePrefix, channelPath(portID, channelID))
}

func ChannelUpgradeErrorKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s/%s", KeyChannelUpgradePrefix, KeyUpgradeErrorPrefix, channelPath(portID, channelID))
}

func ChannelCounterpartyUpgradeKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s/%s", KeyChannelUpgradePrefix, KeyCounterpartyUpgrade, channelPath(portID, channelID))
}

func channelPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", host.KeyPortPrefix, portID, host.KeyChannelPrefix, channelID)
}

// MigrateStore migrates the channel store to the ibc-go v10 store by:
// - Removing channel upgrade sequences
// - Removing any channel upgrade info (i.e. upgrades, counterparty upgrades, upgrade errors)
// - Removing channel params
// - Removing pruning sequences
// NOTE: This migration will fail if any channels are in the FLUSHING or FLUSHCOMPLETE state.
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
			return errors.New("channel in state FLUSHING or FLUSHCOMPLETE found, to proceed with migration, please ensure no channels are currently upgrading")
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
