package types

import (
	"encoding/json"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
)

type ExportMetadataPayload struct {
	ExportMetadata ExportMetadataInnerPayload `json:"export_metadata"`
}

type ExportMetadataInnerPayload struct{}

// ExportMetadata is a no-op since wasm client does not store any metadata in client store
func (c ClientState) ExportMetadata(store sdk.KVStore) []exported.GenesisMetadata {
	payload := ExportMetadataPayload{}

	encodedData, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	ctx := sdk.NewContext(nil, tmproto.Header{Height: 1, Time: time.Now()}, true, nil) // context with infinite gas meter
	response, err := queryContractWithStore(c.CodeId, ctx, store, encodedData)
	if err != nil {
		panic(err)
	}

	output := queryResponse{}
	if err := json.Unmarshal(response, &output); err != nil {
		panic(err)
	}

	genesisMetadata := make([]exported.GenesisMetadata, len(output.GenesisMetadata))
	for i, metadata := range output.GenesisMetadata {
		genesisMetadata[i] = metadata
	}

	return genesisMetadata
}
