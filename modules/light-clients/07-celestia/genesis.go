package celestia

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
)

// ExportMetadata implements exported.ClientState.
func (*ClientState) ExportMetadata(clientStore storetypes.KVStore) []exported.GenesisMetadata {
	panic("unimplemented")
}
