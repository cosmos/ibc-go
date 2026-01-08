package types

import (
	"encoding/json"
	"strings"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec/unknownproto"

	callbacktypes "github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

var (
	_ ibcexported.PacketData         = (*GMPPacketData)(nil)
	_ ibcexported.PacketDataProvider = (*GMPPacketData)(nil)
)

const (
	EncodingJSON     = "application/json"
	EncodingProtobuf = "application/x-protobuf"
	EncodingABI      = "application/x-solidity-abi"
)

// NewGMPPacketData creates a new GMPPacketData instance with the provided parameters.
func NewGMPPacketData(
	sender, receiver string, salt, payload []byte, memo string,
) GMPPacketData {
	return GMPPacketData{
		Sender:   sender,
		Receiver: receiver,
		Salt:     salt,
		Payload:  payload,
		Memo:     memo,
	}
}

func (p GMPPacketData) ValidateBasic() error {
	if strings.TrimSpace(p.Sender) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "missing sender address")
	}
	// receiver is allowed to be empty
	if len(p.Receiver) > MaximumReceiverLength {
		return errorsmod.Wrapf(ibcerrors.ErrInvalidAddress, "recipient address must not exceed %d bytes", MaximumReceiverLength)
	}
	if len(p.Payload) > MaximumPayloadLength {
		return errorsmod.Wrapf(ErrInvalidPayload, "payload must not exceed %d bytes", MaximumPayloadLength)
	}
	if len(p.Salt) > MaximumSaltLength {
		return errorsmod.Wrapf(ErrInvalidSalt, "salt must not exceed %d bytes", MaximumSaltLength)
	}
	if len(p.Memo) > MaximumMemoLength {
		return errorsmod.Wrapf(ErrInvalidMemo, "memo must not exceed %d bytes", MaximumMemoLength)
	}

	return nil
}

// MarshalPacketData attempts to marshal the provided GMPPacketData into bytes with the provided encoding.
func MarshalPacketData(data *GMPPacketData, ics27Version string, encoding string) ([]byte, error) {
	if ics27Version != Version {
		return nil, errorsmod.Wrapf(ErrInvalidVersion, "unsupported ics27 version: %s", ics27Version)
	}

	switch encoding {
	case EncodingJSON:
		return json.Marshal(data)
	case EncodingProtobuf:
		return proto.Marshal(data)
	case EncodingABI:
		return EncodeABIGMPPacketData(data)
	default:
		return nil, errorsmod.Wrapf(ErrInvalidEncoding, "invalid encoding provided, must be either empty or one of [%q, %q, %q], got %s", EncodingJSON, EncodingProtobuf, EncodingABI, encoding)
	}
}

// UnmarshalPacketData attempts to unmarshal the provided bytes into a GMPPacketData with the provided encoding.
func UnmarshalPacketData(bz []byte, ics27Version string, encoding string) (*GMPPacketData, error) {
	if ics27Version != Version {
		return nil, errorsmod.Wrapf(ErrInvalidVersion, "unsupported ics27 version: %s", ics27Version)
	}

	data := &GMPPacketData{}
	switch encoding {
	case EncodingJSON:
		if err := json.Unmarshal(bz, data); err != nil {
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
		data, err = DecodeABIGMPPacketData(bz)
		if err != nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "failed to unmarshal ABI packet data: %s", err)
		}
	default:
		return nil, errorsmod.Wrapf(ErrInvalidEncoding, "invalid encoding provided, must be either empty or one of [%q, %q, %q], got %s", EncodingJSON, EncodingProtobuf, EncodingABI, encoding)
	}

	return data, nil
}

// GetPacketSender returns the sender address of the packet data.
// NOTE:
//   - The sender address is set by the module which requested the packet to be sent,
//     and this module may not have validated the sender address by a signature check.
//   - The sender address must only be used by modules on the sending chain.
//   - sourcePortID is not used in this implementation.
func (p GMPPacketData) GetPacketSender(sourcePortID string) string {
	return p.Sender
}

// GetCustomPacketData returns callback data for the callbacks middleware.
// For "src_callback", returns sender as callback address (auto-registration).
// For other keys, parses memo as JSON and returns the value for the given key.
func (p GMPPacketData) GetCustomPacketData(key string) any {
	if key == callbacktypes.SourceCallbackKey {
		return map[string]any{
			callbacktypes.CallbackAddressKey: p.Sender,
		}
	}

	if len(p.Memo) == 0 {
		return nil
	}

	jsonObject := make(map[string]any)
	if err := json.Unmarshal([]byte(p.Memo), &jsonObject); err != nil {
		return nil
	}

	return jsonObject[key]
}
