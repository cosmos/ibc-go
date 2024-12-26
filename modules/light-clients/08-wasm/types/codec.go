package types

import (
	coreregistry "cosmossdk.io/core/registry"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// RegisterInterfaces registers the Wasm concrete client-related
// implementations and interfaces.
func RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	registry.RegisterImplementations(
		(*exported.ClientState)(nil),
		&ClientState{},
	)
	registry.RegisterImplementations(
		(*exported.ConsensusState)(nil),
		&ConsensusState{},
	)
	registry.RegisterImplementations(
		(*exported.ClientMessage)(nil),
		&ClientMessage{},
	)
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgStoreCode{},
		&MsgMigrateContract{},
		&MsgRemoveChecksum{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
