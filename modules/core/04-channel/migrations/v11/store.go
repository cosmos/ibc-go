package v11

import (
	"fmt"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clientv2keeper "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/keeper"
	chankeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	"github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	chanv2keeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/keeper"
)

const (
	KeyNextSeqSendPrefix = "nextSequenceSend"
	KeyChannelEndPrefix  = "channelEnds"
	KeyChannelPrefix     = "channels"
	KeyPortPrefix        = "ports"
)

// NextSequenceSendV1Key returns the store key for the send sequence of a particular
// channel binded to a specific port.
func NextSequenceSendKey(portID, channelID string) []byte {
	return fmt.Appendf(nil, "%s/%s", KeyNextSeqSendPrefix, channelPath(portID, channelID))
}

func channelPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s/%s", KeyPortPrefix, portID, KeyChannelPrefix, channelID)
}

// MigrateStore migrates the channel store to add support for IBC v2
// for all OPEN UNORDERED channels by:
// - Adding client counterparty information keyed to the channel ID
// - Migrating the NextSequenceSend path to use the v2 format
// - Store an alias key mapping the v1 channel ID to the underlying client ID
func MigrateStore(ctx sdk.Context, storeService corestore.KVStoreService, cdc codec.BinaryCodec,
	channelKeeper *chankeeper.Keeper, chanv2Keeper *chanv2keeper.Keeper, clientv2Keeper *clientv2keeper.Keeper) error {
	store := storeService.OpenKVStore(ctx)

	channelKeeper.IterateChannels(ctx, func(ic types.IdentifiedChannel) (stop bool) {
		// only add counterparty for channels that are OPEN and UNORDERED
		counterparty, ok := channelKeeper.GetV2Counterparty(ctx, ic.PortId, ic.ChannelId)
		if ok {
			clientv2Keeper.SetClientCounterparty(ctx, ic.ChannelId, counterparty)
		}

		// migrate the NextSequenceSend key to the v2 format for every channel
		seqbz, err := store.Get(NextSequenceSendKey(ic.PortId, ic.ChannelId))
		if err != nil {
			panic("NextSequenceSend not found for channel " + ic.ChannelId)
		}
		seq := sdk.BigEndianToUint64(seqbz)
		// set the NextSequenceSend in the v2 keeper
		chanv2Keeper.SetNextSequenceSend(ctx, ic.ChannelId, seq)
		// remove the old NextSequenceSend key
		if err := store.Delete(NextSequenceSendKey(ic.PortId, ic.ChannelId)); err != nil {
			panic("failed to delete NextSequenceSend key for channel " + ic.ChannelId)
		}

		return false
	})
	return nil
}
