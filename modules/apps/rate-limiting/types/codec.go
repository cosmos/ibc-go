package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

// RegisterLegacyAminoCodec registers the necessary rate-limiting interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// Will be implemented with concrete message types
}

// RegisterInterfaces registers the rate-limiting interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// Register messages
	// Will be implemented with concrete message types

	// Register implementations

	// Uncomment when proto messages are defined
	// msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// ModuleCdc references the global rate-limiting module codec. Note, the codec should
// ONLY be used in certain instances of tests and for JSON encoding.
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
