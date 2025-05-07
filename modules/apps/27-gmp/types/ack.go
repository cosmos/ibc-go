package types

import (
	"encoding/json"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec/unknownproto"

	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

// NewAcknowledgement creates a new Acknowledgement
func NewAcknowledgement(result []byte) Acknowledgement {
	return Acknowledgement{
		Result: result,
	}
}

// ValidateBasic performs basic validation on the Acknowledgement
func (Acknowledgement) ValidateBasic() error {
	return nil
}

func MarshalAcknowledgement(data *Acknowledgement, ics27Version string, encoding string) ([]byte, error) {
	if ics27Version != Version {
		panic("unsupported ics27 version")
	}

	switch encoding {
	case EncodingJSON:
		return json.Marshal(data)
	case EncodingProtobuf:
		return proto.Marshal(data)
	case EncodingABI:
		return EncodeABIAcknowledgement(data)
	default:
		return nil, errorsmod.Wrapf(ErrInvalidEncoding, "invalid encoding provided, must be either empty or one of [%q, %q, %q], got %s", EncodingJSON, EncodingProtobuf, EncodingABI, encoding)
	}
}

func UnmarshalAcknowledgement(bz []byte, ics27Version string, encoding string) (*Acknowledgement, error) {
	if ics27Version != Version {
		panic("unsupported ics27 version")
	}

	var data *Acknowledgement
	switch encoding {
	case EncodingJSON:
		if err := json.Unmarshal(bz, &data); err != nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to unmarshal json packet data: %s", err)
		}
	case EncodingProtobuf:
		if err := unknownproto.RejectUnknownFieldsStrict(bz, data, unknownproto.DefaultAnyResolver{}); err != nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to unmarshal protobuf packet data: %s", err)
		}

		if err := proto.Unmarshal(bz, data); err != nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to unmarshal protobuf packet data: %s", err)
		}
	case EncodingABI:
		var err error
		data, err = DecodeABIAcknowledgement(bz)
		if err != nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to unmarshal ABI packet data: %s", err)
		}
	default:
		return nil, errorsmod.Wrapf(ErrInvalidEncoding, "invalid encoding provided, must be either empty or one of [%q, %q, %q], got %s", EncodingJSON, EncodingProtobuf, EncodingABI, encoding)
	}

	return data, nil
}
