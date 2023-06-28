package types

import (
	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// ModuleCdc references the global interchain accounts module codec. Note, the codec
// should ONLY be used in certain instances of tests and for JSON encoding.
//
// The actual codec used for serialization should be provided to interchain accounts and
// defined at the application level.
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// RegisterInterfaces registers the interchain accounts controller types and the concrete InterchainAccount implementation
// against the associated x/auth AccountI and GenesisAccount interfaces.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*authtypes.AccountI)(nil), &InterchainAccount{})
	registry.RegisterImplementations((*authtypes.GenesisAccount)(nil), &InterchainAccount{})
}

// SerializeCosmosTx serializes a slice of sdk.Msg's using the CosmosTx type. The sdk.Msg's are
// packed into Any's and inserted into the Messages field of a CosmosTx. The proto marshaled CosmosTx
// bytes are returned. Only the ProtoCodec is supported for serializing messages.
func SerializeCosmosTx(cdc codec.BinaryCodec, msgs []proto.Message, encoding string) ([]byte, error) {
	// ProtoCodec must be supported
	if _, ok := cdc.(*codec.ProtoCodec); !ok {
		return nil, errorsmod.Wrap(ErrInvalidCodec, "ProtoCodec must be supported for receiving messages on the host chain")
	}

	var bz []byte
	var err error

	msgAnys := make([]*codectypes.Any, len(msgs))

	for i, msg := range msgs {
		msgAnys[i], err = codectypes.NewAnyWithValue(msg)
		if err != nil {
			return nil, err
		}
	}

	cosmosTx := &CosmosTx{
		Messages: msgAnys,
	}

	switch encoding {
	case EncodingProtobuf:
		bz, err = cdc.Marshal(cosmosTx)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "cannot marshal cosmosTx with protobuf")
		}
	case EncodingProto3JSON:
		bz, err = cdc.(*codec.ProtoCodec).MarshalJSON(cosmosTx)
		if err != nil {
			return nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot marshal cosmosTx with json")
		}
	default:
		return nil, errorsmod.Wrapf(ErrInvalidCodec, "unsupported encoding format %s", encoding)
	}

	return bz, nil
}

// DeserializeCosmosTx unmarshals and unpacks a slice of transaction bytes into a slice of sdk.Msg's.
func DeserializeCosmosTx(cdc codec.Codec, data []byte, encoding string) ([]sdk.Msg, error) {
	// ProtoCodec must be supported
	if _, ok := cdc.(*codec.ProtoCodec); !ok {
		return nil, errorsmod.Wrap(ErrInvalidCodec, "ProtoCodec must be supported for receiving messages on the host chain")
	}

	var cosmosTx CosmosTx

	switch encoding {
	case EncodingProtobuf:
		if err := cdc.Unmarshal(data, &cosmosTx); err != nil {
			return nil, errorsmod.Wrapf(err, "cannot unmarshal cosmosTx with protobuf")
		}
	case EncodingProto3JSON:
		if err := cdc.UnmarshalJSON(data, &cosmosTx); err != nil {
			return nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal cosmosTx with json")
		}
	default:
		return nil, errorsmod.Wrapf(ErrInvalidCodec, "unsupported encoding format %s", encoding)
	}

	msgs := make([]sdk.Msg, len(cosmosTx.Messages))

	for i, protoAny := range cosmosTx.Messages {
		var msg sdk.Msg
		err := cdc.UnpackAny(protoAny, &msg)
		if err != nil {
			return nil, err
		}
		msgs[i] = msg
	}

	return msgs, nil
}
