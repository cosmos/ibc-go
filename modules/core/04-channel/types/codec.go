package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	proto "github.com/gogo/protobuf/proto"

	"github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// RegisterInterfaces register the ibc channel submodule interfaces to protobuf
// Any.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterInterface(
		"ibc.core.channel.v1.ChannelI",
		(*exported.ChannelI)(nil),
		&Channel{},
	)
	registry.RegisterInterface(
		"ibc.core.channel.v1.CounterpartyChannelI",
		(*exported.CounterpartyChannelI)(nil),
		&Counterparty{},
	)
	registry.RegisterInterface(
		"ibc.core.channel.v1.PacketI",
		(*exported.PacketI)(nil),
		&Packet{},
	)
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgChannelOpenInit{},
		&MsgChannelOpenTry{},
		&MsgChannelOpenAck{},
		&MsgChannelOpenConfirm{},
		&MsgChannelCloseInit{},
		&MsgChannelCloseConfirm{},
		&MsgRecvPacket{},
		&MsgAcknowledgement{},
		&MsgTimeout{},
		&MsgTimeoutOnClose{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// SubModuleCdc references the global x/ibc/core/04-channel module codec. Note, the codec should
// ONLY be used in certain instances of tests and for JSON encoding.
//
// The actual codec used for serialization should be provided to x/ibc/core/04-channel and
// defined at the application level.
var SubModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// UnpackAcknowledgement unpacks an Any into an Acknowledgement. It returns an error if the
// Any can't be unpacked into an Acknowledgement.
func UnpackAcknowledgement(any *codectypes.Any) (exported.Acknowledgement, error) {
	if any == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnpackAny, "protobuf Any message cannot be nil")
	}

	ack, ok := any.GetCachedValue().(exported.Acknowledgement)
	if !ok {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnpackAny, "cannot unpack Any into Acknowledgement %T", any)
	}

	return ack, nil
}

// PackAcknowledgement constructs a new Any packed with the given acknowledgement value. It returns
// an error if the acknowledgement can't be casted to a protobuf message or if the concrete
// implemention is not registered to the protobuf codec.
func PackAcknowledgement(acknowledgement exported.Acknowledgement) (*codectypes.Any, error) {
	msg, ok := acknowledgement.(proto.Message)
	if !ok {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrPackAny, "cannot proto marshal %T", acknowledgement)
	}

	anyAcknowledgement, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrPackAny, err.Error())
	}

	return anyAcknowledgement, nil
}
