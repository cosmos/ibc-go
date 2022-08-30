package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	abci "github.com/tendermint/tendermint/abci/types"
)

// RegisterLegacyAminoCodec registers the necessary x/ibc interchain query interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgQuery{}, "cosmos-sdk/MsgQuery", nil)
}

// RegisterInterfaces register the ibc transfer module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil), &MsgQuery{})

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global interchain queries module codec. Note, the codec
	// should ONLY be used in certain instances of tests and for JSON encoding.
	//
	// The actual codec used for serialization should be provided to interchain queries and
	// defined at the application level.
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// AminoCdc is a amino codec created to support amino json compatible msgs.
	AminoCdc = codec.NewAminoCodec(amino)
)

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

func init() {
	RegisterLegacyAminoCodec(amino)
	amino.Seal()
}
