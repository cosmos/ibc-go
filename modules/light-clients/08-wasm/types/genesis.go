package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	tm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

func (c ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	gm := make([]exported.GenesisMetadata, 0)
	tm.IterateConsensusMetadata(store, func(key, val []byte) bool {
		gm = append(gm, clienttypes.NewGenesisMetadata(key, val))
		return false
	})
	if len(gm) == 0 {
		return nil
	}
	return gm
	
}