package types

import (
	coreregistry "cosmossdk.io/core/registry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterInterfaces registers the client interfaces to protobuf Any.
func RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgRegisterCounterparty{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
