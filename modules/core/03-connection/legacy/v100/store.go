package v100

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/modules/core/03-connection/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

// MigrateStore performs in-place store migrations from SDK v0.40 of the IBC module to v1.0.0 of ibc-go.
// The migration includes:
//
// - Pruning all connections whose client has been removed (solo machines)
func MigrateStore(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec) (err error) {
	var connections []types.IdentifiedConnection

	// clients and connections use the same store key
	store := ctx.KVStore(storeKey)
	iterator := sdk.KVStorePrefixIterator(store, []byte(host.KeyConnectionPrefix))

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var connection types.ConnectionEnd
		cdc.MustUnmarshal(iterator.Value(), &connection)

		bz := store.Get(host.FullClientStateKey(connection.ClientId))
		if bz == nil {
			// client has been pruned, remove connection as well
			connectionID := host.MustParseConnectionPath(string(iterator.Key()))
			connections = append(connections, types.NewIdentifiedConnection(connectionID, connection))
		}

	}

	for _, conn := range connections {
		store.Delete(host.ConnectionKey(conn.Id))
	}

	return nil
}
