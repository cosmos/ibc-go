package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// ExportMetadata is a no-op since wasm client does not store any metadata in client store
func (c ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	return nil
}