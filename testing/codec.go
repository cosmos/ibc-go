package ibctesting

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/gogoproto/proto"

	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
)

// toJSONAny converts (proto)Any to JSONAny and extracts the json bytes (recursively).
func ToJSONAny(cdc codec.BinaryCodec, protoAny *codectypes.Any) (*icatypes.JSONAny, []byte, error) {
	var message proto.Message

	cdc.UnpackAny(protoAny, &message)

	// Marshal the map back to a byte slice. This function marshalls recursively.
	JSONAnyValue, err := cdc.(*codec.ProtoCodec).MarshalInterfaceJSON(message)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(err, "cannot marshal modified message to bytes")
	}

	result := &icatypes.JSONAny{
		TypeURL: protoAny.TypeUrl,
		Value:   JSONAnyValue,
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return nil, nil, errorsmod.Wrapf(err, "cannot marshal modified json back to bytes")
	}

	return result, bytes, nil
}