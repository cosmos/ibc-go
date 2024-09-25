package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgSendPacket{},
		&MsgRecvPacket{},
		&MsgAcknowledgement{},
		&MsgTimeout{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_PacketMsg_serviceDesc)
}
