package types

import (
	coreregistry "cosmossdk.io/core/registry"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterInterfaces register the ibc channel submodule interfaces to protobuf
// Any.
func RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgSendPacket{},
		&MsgRecvPacket{},
		&MsgTimeout{},
		&MsgAcknowledgement{},
	)
}
