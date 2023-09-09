package types

import (
	"time"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// NewGenesisState creates an 08-wasm GenesisState instance.
func NewGenesisState(contracts []Contract) *GenesisState {
	return &GenesisState{Contracts: contracts}
}

// ExportMetadata exports all the consensus metadata in the client store so they
// can be included in clients genesis and imported by a ClientKeeper
func (cs ClientState) ExportMetadata(store storetypes.KVStore) []exported.GenesisMetadata {
	payload := queryMsg{
		ExportMetadata: &exportMetadataMsg{},
	}

	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: time.Now()}, true, nil) // context with infinite gas meter
	result, err := wasmQuery[exportMetadataResult](ctx, store, &cs, payload)
	if err != nil {
		panic(err)
	}

	genesisMetadata := make([]exported.GenesisMetadata, len(result.GenesisMetadata))
	for i, metadata := range result.GenesisMetadata {
		genesisMetadata[i] = metadata
	}

	return genesisMetadata
}
