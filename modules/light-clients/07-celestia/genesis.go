package celestia

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// ExportMetadata implements exported.ClientState.
func (cs *ClientState) ExportMetadata(clientStore storetypes.KVStore) []exported.GenesisMetadata {
	return cs.BaseClient.ExportMetadata(clientStore)
}
