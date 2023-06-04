package types

import (
	"encoding/json"
	"reflect"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
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

// JSONAny is used to serialize and deserialize messages in the Any type for json encoding.
type JSONAny struct {
	TypeURL string `json:"type_url,omitempty"`
	Value   []byte `json:"value,omitempty"`
}

// JSONCosmosTx is used to serialize and deserialize messages in the CosmosTx type for json encoding.
type JSONCosmosTx struct {
	Messages []*JSONAny `json:"messages,omitempty"`
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
		msgAnys := make([]*JSONAny, len(msgs))
		for i, msg := range msgs {
			jsonValue, err := cdc.(*codec.ProtoCodec).MarshalJSON(msg)
			if err != nil {
				return nil, err
			}
			msgAnys[i] = &JSONAny{
				TypeURL: "/" + proto.MessageName(msg),
				Value:   jsonValue,
			}

			cosmosTx := JSONCosmosTx{
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
		var cosmosTx JSONCosmosTx
		if err := json.Unmarshal(data, &cosmosTx); err != nil {
			return nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal cosmosTx with json")
		}

		msgs = make([]sdk.Msg, len(cosmosTx.Messages))

		for i, jsonAny := range cosmosTx.Messages {
			_, message, err := extractJsonAny(cdc, jsonAny)
			if err != nil {
				return nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal the %d-th json message: %s", i, string(jsonAny.Value))
			}

			msg, ok := message.(sdk.Msg)
			if !ok {
				return nil, errorsmod.Wrapf(ErrUnsupported, "message %T does not implement sdk.Msg", message)
			}
			msgs[i] = msg
		}
	default:
		return nil, errorsmod.Wrapf(ErrUnsupportedEncoding, "encoding type %s is not supported", encoding)
	}

	return msgs, nil
}

// extractJsonAny converts JSONAny to (proto)Any and extracts the proto.Message (recursively).
func extractJsonAny(cdc codec.BinaryCodec, jsonAny *JSONAny) (*codectypes.Any, proto.Message, error) {
	// get the type_url field
	typeURL := jsonAny.TypeURL
	// get uninitialized proto.Message
	message, err := cdc.(*codec.ProtoCodec).InterfaceRegistry().Resolve(typeURL)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(err, "cannot resolve this typeURL to a proto.Message: %s", typeURL)
	}

	value := jsonAny.Value
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(value, &jsonMap); err != nil {
		return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal value to json")
	}

	// Check if message has Any fields
	val := reflect.ValueOf(message).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := field.Type()
		fieldJSONName := val.Type().Field(i).Tag.Get("json")
		// Remove ,omitempty if it's present
		fieldJSONName = strings.Split(fieldJSONName, ",")[0]

		if fieldType == reflect.TypeOf((*codectypes.Any)(nil)) {
			// get the any field
			subJsonAnyMap, ok := jsonMap[fieldJSONName].(map[string]interface{})
			if !ok {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot assert the any field to map[string]interface{}")
			}

			// Create the JSONAny
			jsonBytes, err := json.Marshal(subJsonAnyMap)
			if err != nil {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot marshal the any field to bytes")
			}
			subJsonAny := &JSONAny{}
			if err = json.Unmarshal(jsonBytes, subJsonAny); err != nil {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal the any field with json")
			}

			protoAny, _, err := extractJsonAny(cdc, subJsonAny)
			if err != nil {
				return nil, nil, err
			}

			// Set back the new value
			field.Set(reflect.ValueOf(protoAny))
			// Remove this field from jsonAnyMap
			delete(jsonMap, fieldJSONName)
		}
	}

	// Marshal the map back to a byte slice
	modifiedJsonAnyValue, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(err, "cannot marshal modified json map back to bytes")
	}

	if err = cdc.(*codec.ProtoCodec).UnmarshalJSON(modifiedJsonAnyValue, message); err != nil {
		return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal the json message")
	}

	result, err := codectypes.NewAnyWithValue(message)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(err, "cannot pack the message into Any")
	}

	return result, message, nil
}
