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
			protoAny, err := codectypes.NewAnyWithValue(msg)
			if err != nil {
				return nil, err
			}
			jsonAny, _, err := toJSONAny(cdc, protoAny)
			if err != nil {
				return nil, err
			}
			msgAnys[i] = jsonAny
		}

		cosmosTx := &JSONCosmosTx{
			Messages: msgAnys,
		}

		bz, err = json.Marshal(cosmosTx)
		if err != nil {
			return nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot marshal cosmosTx with json")
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
			_, message, err := fromJSONAny(cdc, jsonAny)
			if err != nil {
				return nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal the %d-th json message: %s", i, string(jsonAny.Value))
			}

			msg, ok := message.(sdk.Msg)
			if !ok {
				return nil, errorsmod.Wrapf(ErrUnknownDataType, "message %T does not implement sdk.Msg", message)
			}
			msgs[i] = msg
		}
	default:
		return nil, errorsmod.Wrapf(ErrUnsupportedEncoding, "encoding type %s is not supported", encoding)
	}

	return msgs, nil
}

// fromJSONAny converts JSONAny to (proto)Any and extracts the proto.Message (recursively).
func fromJSONAny(cdc codec.BinaryCodec, jsonAny *JSONAny) (*codectypes.Any, proto.Message, error) {
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
		fieldJSONName, ok := val.Type().Field(i).Tag.Lookup("json")
		if !ok {
			return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot get the json tag of the field")
		}
		// Remove ,omitempty if it's present
		fieldJSONName = strings.Split(fieldJSONName, ",")[0]

		if fieldType == reflect.TypeOf((*codectypes.Any)(nil)) {
			// get the any field
			subJSONAnyMap, ok := jsonMap[fieldJSONName].(map[string]interface{})
			if !ok {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot assert the any field to map[string]interface{}")
			}

			// Create the JSONAny
			jsonBytes, err := json.Marshal(subJSONAnyMap)
			if err != nil {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot marshal the any field to bytes")
			}
			subJSONAny := &JSONAny{}
			if err = json.Unmarshal(jsonBytes, subJSONAny); err != nil {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal the any field with json")
			}

			protoAny, _, err := fromJSONAny(cdc, subJSONAny)
			if err != nil {
				return nil, nil, err
			}

			// Set back the new value
			field.Set(reflect.ValueOf(protoAny))
			// Remove this field from jsonAnyMap
			delete(jsonMap, fieldJSONName)
		} else if fieldType.Kind() == reflect.Slice && fieldType.Elem() == reflect.TypeOf((*codectypes.Any)(nil)) {
			sliceSubJSONAnyMap, ok := jsonMap[fieldJSONName].([]interface{})
			if !ok {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot assert the slice of any field to []interface{}")
			}

			protoAnys := make([]*codectypes.Any, len(sliceSubJSONAnyMap))

			for i, subJSONAnyInterface := range sliceSubJSONAnyMap {
				subJSONAnyMap, ok := subJSONAnyInterface.(map[string]interface{})
				if !ok {
					return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot assert the any field to map[string]interface{}")
				}

				// Create the JSONAny
				jsonBytes, err := json.Marshal(subJSONAnyMap)
				if err != nil {
					return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot marshal the any field to bytes")
				}
				subJSONAny := &JSONAny{}
				if err = json.Unmarshal(jsonBytes, subJSONAny); err != nil {
					return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal the any field with json")
				}

				protoAny, _, err := fromJSONAny(cdc, subJSONAny)
				if err != nil {
					return nil, nil, err
				}

				protoAnys[i] = protoAny
			}

			field.Set(reflect.ValueOf(protoAnys))
			delete(jsonMap, fieldJSONName)
		}
	}

	// Marshal the map back to a byte slice
	modifiedJSONAnyValue, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot marshal modified json back to bytes")
	}

	if err = cdc.(*codec.ProtoCodec).UnmarshalJSON(modifiedJSONAnyValue, message); err != nil {
		return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot unmarshal modified json to message")
	}

	result, err := codectypes.NewAnyWithValue(message)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(err, "cannot pack the message into Any")
	}

	return result, message, nil
}

// toJSONAny converts (proto)Any to JSONAny and extracts the json bytes (recursively).
func toJSONAny(cdc codec.BinaryCodec, protoAny *codectypes.Any) (*JSONAny, []byte, error) {
	var message proto.Message

	err := cdc.UnpackAny(protoAny, &message)
	if err != nil {
		return nil, nil, err
	}

	messageMap := make(map[string]interface{})

	val := reflect.ValueOf(message).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := field.Type()
		fieldJSONName, ok := val.Type().Field(i).Tag.Lookup("json")
		if !ok {
			return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot get the json tag of the field")
		}
		// Remove ,omitempty if it's present
		fieldJSONName = strings.Split(fieldJSONName, ",")[0]

		switch {
		case fieldType == reflect.TypeOf((*codectypes.Any)(nil)):
			subProtoAny, ok := field.Interface().(*codectypes.Any)
			if !ok {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot assert the any field to *codectypes.Any")
			}

			subJSONAny, _, err := toJSONAny(cdc, subProtoAny)
			if err != nil {
				return nil, nil, err
			}

			messageMap[fieldJSONName] = subJSONAny
		case fieldType.Kind() == reflect.Slice && fieldType.Elem() == reflect.TypeOf((*codectypes.Any)(nil)):
			subProtoAnys, ok := field.Interface().([]*codectypes.Any)
			if !ok {
				return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot assert the slice of any field to []*codectypes.Any")
			}

			subJSONAnys := make([]*JSONAny, len(subProtoAnys))

			for i, subProtoAny := range subProtoAnys {
				subJSONAny, _, err := toJSONAny(cdc, subProtoAny)
				if err != nil {
					return nil, nil, err
				}

				subJSONAnys[i] = subJSONAny
			}

			messageMap[fieldJSONName] = subJSONAnys
		default:
			messageMap[fieldJSONName] = field.Interface()
		}
	}

	// Marshal the map back to a byte slice. This function marshalls recursively.
	JSONAnyValue, err := json.Marshal(messageMap)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(ErrUnknownDataType, "cannot marshal modified message to bytes")
	}

	result := &JSONAny{
		TypeURL: protoAny.TypeUrl,
		Value:   JSONAnyValue,
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(err, "cannot marshal modified json back to bytes")
	}

	return result, bytes, nil
}
