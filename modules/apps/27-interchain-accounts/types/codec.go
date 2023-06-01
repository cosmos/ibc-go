package types

import (
	"bytes"
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/gogo/protobuf/jsonpb"
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

// JsonAny is used to serialize and deserialize messages in the Any type for json encoding.
type JsonAny struct {
	TypeUrl string `json:"type_url,omitempty"`
	Value   []byte `json:"value,omitempty"`
}

// JsonCosmosTx is used to serialize and deserialize messages in the CosmosTx type for json encoding.
type JsonCosmosTx struct {
	Messages []*JsonAny `json:"messages,omitempty"`
}

// SerializeCosmosTx serializes a slice of sdk.Msg's using the CosmosTx type. The sdk.Msg's are
// packed into Any's and inserted into the Messages field of a CosmosTx. The proto marshaled CosmosTx
// bytes are returned. Only the ProtoCodec is supported for serializing messages.
func SerializeCosmosTx(cdc codec.BinaryCodec, msgs []proto.Message, encoding string) ([]byte, error) {
	// only ProtoCodec is supported
	if _, ok := cdc.(*codec.ProtoCodec); !ok {
		return nil, errorsmod.Wrap(ErrInvalidCodec, "only ProtoCodec is supported for receiving messages on the host chain")
	}

	var bz []byte
	var err error

	switch encoding {
	case EncodingProtobuf:
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
	
		bz, err = cdc.Marshal(cosmosTx)
		if err != nil {
			return nil, err
		}
	case EncodingJSON:
		msgAnys := make([]*JsonAny, len(msgs))
		for i, msg := range msgs {
			jsonValue, err := cdc.(*codec.ProtoCodec).MarshalJSON(msg)
			if err != nil {
				return nil, err
			}
			msgAnys[i] = &JsonAny{
				TypeUrl:     "/" + proto.MessageName(msg),
				Value:       jsonValue,
			}

			cosmosTx := JsonCosmosTx{
				Messages: msgAnys,
			}

			bz, err = json.Marshal(cosmosTx)
			if err != nil {
				return nil, err
			}
		}
	default:
		return nil, errorsmod.Wrapf(ErrUnsupportedEncoding, "encoding type %s is not supported", encoding)
	}

	return bz, nil
}

// DeserializeCosmosTx unmarshals and unpacks a slice of transaction bytes into a slice of sdk.Msg's.
func DeserializeCosmosTx(cdc codec.BinaryCodec, data []byte, encoding string) ([]sdk.Msg, error) {
	// ProtoCodec must be supported
	if _, ok := cdc.(*codec.ProtoCodec); !ok {
		return nil, errorsmod.Wrap(ErrInvalidCodec, "ProtoCodec must be supported for receiving messages on the host chain")
	}

	var msgs []sdk.Msg

	switch encoding {
	case EncodingProtobuf:
		var cosmosTx CosmosTx
		if err := cdc.Unmarshal(data, &cosmosTx); err != nil {
			return nil, err
		}

		msgs = make([]sdk.Msg, len(cosmosTx.Messages))

		for i, protoAny := range cosmosTx.Messages {
			var msg sdk.Msg

			err := cdc.UnpackAny(protoAny, &msg)
			if err != nil {
				return nil, err
			}

			msgs[i] = msg
		}
	case EncodingJSON:
		var cosmosTx JsonCosmosTx
		// This case does not rely on cdc to unpack the Any.
		// cdc is only used to access the interface registry.
		interfaceRegistry := cdc.(*codec.ProtoCodec).InterfaceRegistry()
		// this cosmosTx is not the same as the one in the protobuf case
		// its Any needs to be unpacked using json instead of protobuf
		if err := json.Unmarshal(data, &cosmosTx); err != nil {
			// TODO: not sure if this err is indeterminate
			return nil, err
		}

		msgs = make([]sdk.Msg, len(cosmosTx.Messages))

		for i, jsonAny := range cosmosTx.Messages {
			message, err := interfaceRegistry.Resolve(jsonAny.TypeUrl)
			if err != nil {
				return nil, err
			}
			if err = jsonpb.Unmarshal(bytes.NewReader(jsonAny.Value), message); err != nil {
				// TODO: not sure if this err is indeterminate
				return nil, err
			}

			msgs[i] = message.(sdk.Msg)
		}
	default:
		return nil, errorsmod.Wrapf(ErrUnsupportedEncoding, "encoding type %s is not supported", encoding)
	}

	return msgs, nil
}
