package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// ModuleCdc references the global interchain queries module codec. Note, the codec
// should ONLY be used in certain instances of tests and for JSON encoding.
//
// The actual codec used for serialization should be provided to interchain queries and
// defined at the application level.
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

func SerializeCosmosQuery(reqs []abci.RequestQuery) (bz []byte, err error) {
	q := &CosmosQuery{
		Requests: reqs,
	}
	return ModuleCdc.Marshal(q)
}

func DeserializeCosmosQuery(bz []byte) (reqs []abci.RequestQuery, err error) {
	var q CosmosQuery
	err = ModuleCdc.Unmarshal(bz, &q)
	return q.Requests, err
}

func SerializeCosmosResponse(resps []abci.ResponseQuery) (bz []byte, err error) {
	r := &CosmosResponse{
		Responses: resps,
	}
	return ModuleCdc.Marshal(r)
}

func DeserializeCosmosResponse(bz []byte) (resps []abci.ResponseQuery, err error) {
	var r CosmosResponse
	err = ModuleCdc.Unmarshal(bz, &r)
	return r.Responses, err
}
