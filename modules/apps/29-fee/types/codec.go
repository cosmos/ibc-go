package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterInterfaces register the 29-fee module interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	//	registry.RegisterImplementations((*sdk.Msg)(nil), &Msg{})

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
