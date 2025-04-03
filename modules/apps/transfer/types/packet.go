package types

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec/unknownproto"

	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

// InternalTransferRepresentation defines a struct used internally by the transfer application to represent a fungible token transfer
type InternalTransferRepresentation struct {
	// the tokens to be transferred
	Token Token
	// the sender address
	Sender string
	// the recipient address on the destination chain
	Receiver string
	// optional memo
	Memo string
}

var (
	_ ibcexported.PacketData         = (*FungibleTokenPacketData)(nil)
	_ ibcexported.PacketDataProvider = (*FungibleTokenPacketData)(nil)
)

const (
	EncodingJSON     = "application/json"
	EncodingProtobuf = "application/x-protobuf"
	EncodingABI      = "application/x-solidity-abi"
)

// NewFungibleTokenPacketData constructs a new FungibleTokenPacketData instance
func NewFungibleTokenPacketData(
	denom string, amount string,
	sender, receiver string,
	memo string,
) FungibleTokenPacketData {
	return FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		Memo:     memo,
	}
}

// ValidateBasic is used for validating the token transfer.
// NOTE: The addresses formats are not validated as the sender and recipient can have different
// formats defined by their corresponding chains that are not known to IBC.
func (ftpd FungibleTokenPacketData) ValidateBasic() error {
	amount, ok := sdkmath.NewIntFromString(ftpd.Amount)
	if !ok {
		return errorsmod.Wrapf(ErrInvalidAmount, "unable to parse transfer amount (%s) into math.Int", ftpd.Amount)
	}
	if !amount.IsPositive() {
		return errorsmod.Wrapf(ErrInvalidAmount, "amount must be strictly positive: got %d", amount)
	}
	if strings.TrimSpace(ftpd.Sender) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "sender address cannot be blank")
	}
	if strings.TrimSpace(ftpd.Receiver) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "receiver address cannot be blank")
	}
	denom := ExtractDenomFromPath(ftpd.Denom)
	return denom.Validate()
}

// GetBytes is a helper for serialising the packet to bytes.
// The memo field of FungibleTokenPacketData is marked with the JSON omitempty tag
// ensuring that the memo field is not included in the marshalled bytes if one is not specified.
func (ftpd FungibleTokenPacketData) GetBytes() []byte {
	bz, err := json.Marshal(ftpd)
	if err != nil {
		panic(errors.New("cannot marshal FungibleTokenPacketData into bytes"))
	}

	return bz
}

// GetPacketSender returns the sender address embedded in the packet data.
//
// NOTE:
//   - The sender address is set by the module which requested the packet to be sent,
//     and this module may not have validated the sender address by a signature check.
//   - The sender address must only be used by modules on the sending chain.
//   - sourcePortID is not used in this implementation.
func (ftpd FungibleTokenPacketData) GetPacketSender(sourcePortID string) string {
	return ftpd.Sender
}

// GetCustomPacketData interprets the memo field of the packet data as a JSON object
// and returns the value associated with the given key.
// If the key is missing or the memo is not properly formatted, then nil is returned.
func (ftpd FungibleTokenPacketData) GetCustomPacketData(key string) any {
	if len(ftpd.Memo) == 0 {
		return nil
	}

	jsonObject := make(map[string]any)
	err := json.Unmarshal([]byte(ftpd.Memo), &jsonObject)
	if err != nil {
		return nil
	}

	memoData, found := jsonObject[key]
	if !found {
		return nil
	}

	return memoData
}

// NewInternalTransferRepresentation constructs a new InternalTransferRepresentation instance
func NewInternalTransferRepresentation(
	token Token,
	sender, receiver string,
	memo string,
) InternalTransferRepresentation {
	return InternalTransferRepresentation{
		Token:    token,
		Sender:   sender,
		Receiver: receiver,
		Memo:     memo,
	}
}

// ValidateBasic is used for validating the token transfer.
// NOTE: The addresses formats are not validated as the sender and recipient can have different
// formats defined by their corresponding chains that are not known to IBC.
func (ftpd InternalTransferRepresentation) ValidateBasic() error {
	if strings.TrimSpace(ftpd.Sender) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "sender address cannot be blank")
	}

	if strings.TrimSpace(ftpd.Receiver) == "" {
		return errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "receiver address cannot be blank")
	}

	if err := ftpd.Token.Validate(); err != nil {
		return err
	}

	if len(ftpd.Memo) > MaximumMemoLength {
		return errorsmod.Wrapf(ErrInvalidMemo, "memo must not exceed %d bytes", MaximumMemoLength)
	}

	return nil
}

// GetCustomPacketData interprets the memo field of the packet data as a JSON object
// and returns the value associated with the given key.
// If the key is missing or the memo is not properly formatted, then nil is returned.
func (ftpd InternalTransferRepresentation) GetCustomPacketData(key string) any {
	if len(ftpd.Memo) == 0 {
		return nil
	}

	jsonObject := make(map[string]any)
	err := json.Unmarshal([]byte(ftpd.Memo), &jsonObject)
	if err != nil {
		return nil
	}

	memoData, found := jsonObject[key]
	if !found {
		return nil
	}

	return memoData
}

// GetPacketSender returns the sender address embedded in the packet data.
//
// NOTE:
//   - The sender address is set by the module which requested the packet to be sent,
//     and this module may not have validated the sender address by a signature check.
//   - The sender address must only be used by modules on the sending chain.
//   - sourcePortID is not used in this implementation.
func (ftpd InternalTransferRepresentation) GetPacketSender(sourcePortID string) string {
	return ftpd.Sender
}

// MarshalPacketData attempts to marshal the provided FungibleTokenPacketData into bytes with the provided encoding.
func MarshalPacketData(data FungibleTokenPacketData, ics20Version string, encoding string) ([]byte, error) {
	if ics20Version != V1 {
		panic("unsupported ics20 version")
	}

	switch encoding {
	case EncodingJSON:
		return json.Marshal(data)
	case EncodingProtobuf:
		return proto.Marshal(&data)
	case EncodingABI:
		return EncodeABIFungibleTokenPacketData(&data)
	default:
		return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "invalid encoding provided, must be either empty or one of [%q, %q], got %s", EncodingJSON, EncodingProtobuf, encoding)
	}
}

// UnmarshalPacketData attempts to unmarshal the provided packet data bytes into a InternalTransferRepresentation.
func UnmarshalPacketData(bz []byte, ics20Version string, encoding string) (InternalTransferRepresentation, error) {
	const failedUnmarshalingErrorMsg = "cannot unmarshal %s transfer packet data: %s"

	// Depending on the ics20 version, we use a different default encoding (json for V1, proto for V2)
	// and we have a different type to unmarshal the data into.
	var data proto.Message
	switch ics20Version {
	case V1:
		if encoding == "" {
			encoding = EncodingJSON
		}
		data = &FungibleTokenPacketData{}
	default:
		return InternalTransferRepresentation{}, errorsmod.Wrap(ErrInvalidVersion, ics20Version)
	}

	errorMsgVersion := "ICS20-V1"

	// Here we perform the unmarshaling based on the specified encoding.
	// The functions act on the generic "data" variable which is of type proto.Message (an interface).
	switch encoding {
	case EncodingJSON:
		if err := json.Unmarshal(bz, &data); err != nil {
			return InternalTransferRepresentation{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, failedUnmarshalingErrorMsg, errorMsgVersion, err.Error())
		}
	case EncodingProtobuf:
		if err := unknownproto.RejectUnknownFieldsStrict(bz, data, unknownproto.DefaultAnyResolver{}); err != nil {
			return InternalTransferRepresentation{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, failedUnmarshalingErrorMsg, errorMsgVersion, err.Error())
		}

		if err := proto.Unmarshal(bz, data); err != nil {
			return InternalTransferRepresentation{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, failedUnmarshalingErrorMsg, errorMsgVersion, err.Error())
		}
	case EncodingABI:
		if ics20Version != V1 {
			return InternalTransferRepresentation{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "encoding %s is only supported for ICS20-V1", EncodingABI)
		}
		var err error
		data, err = DecodeABIFungibleTokenPacketData(bz)
		if err != nil {
			return InternalTransferRepresentation{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, failedUnmarshalingErrorMsg, errorMsgVersion, err.Error())
		}
	default:
		return InternalTransferRepresentation{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "invalid encoding provided, must be either empty or one of [%q, %q, %q], got %s", EncodingJSON, EncodingProtobuf, EncodingABI, encoding)
	}

	// When the unmarshaling is done, we want to retrieve the underlying data type based on the value of ics20Version
	// Since it has to be v1, we convert the data to FungibleTokenPacketData and then call the conversion function to construct
	// the v2 type.
	datav1, ok := data.(*FungibleTokenPacketData)
	if !ok {
		// We should never get here, as we manually constructed the type at the beginning of the file
		return InternalTransferRepresentation{}, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot convert proto message into FungibleTokenPacketData")
	}
	// The call to ValidateBasic for V1 is done inside PacketDataV1toV2.
	return PacketDataV1ToV2(*datav1)
}

// PacketDataV1ToV2 converts a v1 packet data to a v2 packet data. The packet data is validated
// before conversion.
func PacketDataV1ToV2(packetData FungibleTokenPacketData) (InternalTransferRepresentation, error) {
	if err := packetData.ValidateBasic(); err != nil {
		return InternalTransferRepresentation{}, errorsmod.Wrapf(err, "invalid packet data")
	}

	denom := ExtractDenomFromPath(packetData.Denom)
	return InternalTransferRepresentation{
		Token: Token{
			Denom:  denom,
			Amount: packetData.Amount,
		},
		Sender:   packetData.Sender,
		Receiver: packetData.Receiver,
		Memo:     packetData.Memo,
	}, nil
}
