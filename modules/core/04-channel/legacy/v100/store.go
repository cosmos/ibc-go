package v100

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// MigrateStore performs in-place store migrations from SDK v0.40 of the IBC module to v1.0.0 of ibc-go.
// The migration includes:
//
// - Pruning all channels whose connection has been removed (solo machines)
func MigrateStore(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec) (err error) {
	var channels []types.IdentifiedChannel

	// connections and channels use the same store key
	store := ctx.KVStore(storeKey)

	iterator := sdk.KVStorePrefixIterator(store, []byte(host.KeyChannelEndPrefix))

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var channel types.Channel
		cdc.MustUnmarshal(iterator.Value(), &channel)

		bz := store.Get(host.ConnectionKey(channel.ConnectionHops[0]))
		if bz == nil {
			// connection has been pruned, remove channel as well
			portID, channelID := host.MustParseChannelPath(string(iterator.Key()))
			channels = append(channels, types.NewIdentifiedChannel(portID, channelID, channel))
		}
	}

	for _, channel := range channels {
		store.Delete(host.ChannelKey(channel.PortId, channel.ChannelId))
	}

	return nil
}
