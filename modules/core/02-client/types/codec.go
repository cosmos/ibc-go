package types

import (
	"github.com/cosmos/gogoproto/proto"
	gogoprotoany "github.com/cosmos/gogoproto/types/any"

	coreregistry "cosmossdk.io/core/registry"
	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	"github.com/cosmos/ibc-go/v9/modules/core/exported"
)

// RegisterInterfaces registers the client interfaces to protobuf Any.
func RegisterInterfaces(registry coreregistry.InterfaceRegistrar) {
	registry.RegisterInterface(
		"ibc.core.client.v1.ClientState",
		(*exported.ClientState)(nil),
	)
	registry.RegisterInterface(
		"ibc.core.client.v1.ConsensusState",
		(*exported.ConsensusState)(nil),
	)
	registry.RegisterInterface(
		"ibc.core.client.v1.Header",
		(*exported.ClientMessage)(nil),
	)
	registry.RegisterInterface(
		"ibc.core.client.v1.Height",
		(*exported.Height)(nil),
		&Height{},
	)
	registry.RegisterInterface(
		"ibc.core.client.v1.Misbehaviour",
		(*exported.ClientMessage)(nil),
	)
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgCreateClient{},
		&MsgUpdateClient{},
		&MsgUpgradeClient{},
		&MsgSubmitMisbehaviour{},
		&MsgRecoverClient{},
		&MsgIBCSoftwareUpgrade{},
		&MsgUpdateParams{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// PackClientState constructs a new Any packed with the given client state value. It returns
// an error if the client state can't be casted to a protobuf message or if the concrete
// implementation is not registered to the protobuf codec.
func PackClientState(clientState exported.ClientState) (*gogoprotoany.Any, error) {
	msg, ok := clientState.(proto.Message)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrPackAny, "cannot proto marshal %T", clientState)
	}

	anyClientState, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, errorsmod.Wrap(ibcerrors.ErrPackAny, err.Error())
	}

	return anyClientState, nil
}

// UnpackClientState unpacks an Any into a ClientState. It returns an error if the
// client state can't be unpacked into a ClientState.
func UnpackClientState(protoAny *gogoprotoany.Any) (exported.ClientState, error) {
	if protoAny == nil {
		return nil, errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil")
	}

	clientState, ok := protoAny.GetCachedValue().(exported.ClientState)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnpackAny, "cannot unpack Any into ClientState %T", protoAny)
	}

	return clientState, nil
}

// PackConsensusState constructs a new Any packed with the given consensus state value. It returns
// an error if the consensus state can't be casted to a protobuf message or if the concrete
// implementation is not registered to the protobuf codec.
func PackConsensusState(consensusState exported.ConsensusState) (*gogoprotoany.Any, error) {
	msg, ok := consensusState.(proto.Message)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrPackAny, "cannot proto marshal %T", consensusState)
	}

	anyConsensusState, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, errorsmod.Wrap(ibcerrors.ErrPackAny, err.Error())
	}

	return anyConsensusState, nil
}

// MustPackConsensusState calls PackConsensusState and panics on error.
func MustPackConsensusState(consensusState exported.ConsensusState) *gogoprotoany.Any {
	anyConsensusState, err := PackConsensusState(consensusState)
	if err != nil {
		panic(err)
	}

	return anyConsensusState
}

// UnpackConsensusState unpacks an Any into a ConsensusState. It returns an error if the
// consensus state can't be unpacked into a ConsensusState.
func UnpackConsensusState(protoAny *gogoprotoany.Any) (exported.ConsensusState, error) {
	if protoAny == nil {
		return nil, errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil")
	}

	consensusState, ok := protoAny.GetCachedValue().(exported.ConsensusState)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnpackAny, "cannot unpack Any into ConsensusState %T", protoAny)
	}

	return consensusState, nil
}

// PackClientMessage constructs a new Any packed with the given value. It returns
// an error if the value can't be casted to a protobuf message or if the concrete
// implementation is not registered to the protobuf codec.
func PackClientMessage(clientMessage exported.ClientMessage) (*gogoprotoany.Any, error) {
	msg, ok := clientMessage.(proto.Message)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrPackAny, "cannot proto marshal %T", clientMessage)
	}

	protoAny, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return nil, errorsmod.Wrap(ibcerrors.ErrPackAny, err.Error())
	}

	return protoAny, nil
}

// UnpackClientMessage unpacks an Any into a ClientMessage. It returns an error if the
// consensus state can't be unpacked into a ClientMessage.
func UnpackClientMessage(protoAny *gogoprotoany.Any) (exported.ClientMessage, error) {
	if protoAny == nil {
		return nil, errorsmod.Wrap(ibcerrors.ErrUnpackAny, "protobuf Any message cannot be nil")
	}

	clientMessage, ok := protoAny.GetCachedValue().(exported.ClientMessage)
	if !ok {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnpackAny, "cannot unpack Any into Header %T", protoAny)
	}

	return clientMessage, nil
}
