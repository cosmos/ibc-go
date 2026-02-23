package types

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/cosmos/gogoproto/proto"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
)

// NewAccountIdentifier creates a new AccountIdentifier with the given clientId, sender, and salt.
func NewAccountIdentifier(clientID, sender string, salt []byte) AccountIdentifier {
	return AccountIdentifier{
		ClientId: clientID,
		Sender:   sender,
		Salt:     salt,
	}
}

// NewICS27Account creates a new ICS27Account with the given address and accountId.
func NewICS27Account(addr string, accountID *AccountIdentifier) ICS27Account {
	return ICS27Account{
		Address:   addr,
		AccountId: accountID,
	}
}

// BuildAddressPredictable generates an account address for the gmp module with len = types.AccountAddrLen using the
// Cosmos SDK address.Module function.
// Internally a key is built containing:
// (len(clientId) | clientId | len(sender) | sender | len(salt) | salt).
//
// All method parameter values must be valid and not nil.
//
// This function was copied from wasmd and modified.
// <https://github.com/CosmWasm/wasmd/blob/632fc333d01a84fa5426de6783f7797ad2825e25/x/wasm/keeper/addresses.go#L49>
func BuildAddressPredictable(accountID *AccountIdentifier) (sdk.AccAddress, error) {
	if err := host.ClientIdentifierValidator(accountID.ClientId); err != nil {
		return nil, errorsmod.Wrapf(err, "invalid client ID %s", accountID.ClientId)
	}
	if strings.TrimSpace(accountID.Sender) == "" {
		return nil, errorsmod.Wrap(ibcerrors.ErrInvalidAddress, "missing sender address")
	}

	clientIDBz := uint64LengthPrefix([]byte(accountID.ClientId))
	senderBz := uint64LengthPrefix([]byte(accountID.Sender))
	saltBz := uint64LengthPrefix(accountID.Salt)
	key := make([]byte, len(clientIDBz)+len(senderBz)+len(saltBz))
	copy(key[0:], clientIDBz)
	copy(key[len(clientIDBz):], senderBz)
	copy(key[len(clientIDBz)+len(senderBz):], saltBz)
	return address.Module(accountsKey, key)[:AccountAddrLen], nil
}

// uint64LengthPrefix prepend big endian encoded byte length
func uint64LengthPrefix(bz []byte) []byte {
	return append(sdk.Uint64ToBigEndian(uint64(len(bz))), bz...)
}

// DeserializeCosmosTx unmarshals and unpacks a slice of transaction bytes into a slice of sdk.Msg's.
// The transaction bytes are unmarshaled depending on the encoding type passed in. The sdk.Msg's are
// unpacked from Any's and returned.
func DeserializeCosmosTx(cdc codec.Codec, data []byte) ([]sdk.Msg, error) {
	// this is a defensive check to ensure only the ProtoCodec is used for message deserialization
	if _, ok := cdc.(*codec.ProtoCodec); !ok {
		return nil, errorsmod.Wrap(ErrInvalidCodec, "only the ProtoCodec may be used for receiving messages on the host chain")
	}

	var cosmosTx CosmosTx
	err := cdc.Unmarshal(data, &cosmosTx)
	if err != nil {
		if err := cdc.UnmarshalJSON(data, &cosmosTx); err != nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot unmarshal CosmosTx with protobuf or protojson: %v", err)
		}
		reconstructedData, err := cdc.MarshalJSON(&cosmosTx)
		if err != nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot remarshal CosmosTx with proto3 json: %v", err)
		}
		if isEqual, err := equalJSON(data, reconstructedData); err != nil {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "cannot compare original and reconstructed JSON: %v", err)
		} else if !isEqual {
			return nil, errorsmod.Wrapf(ibcerrors.ErrInvalidType, "original and reconstructed JSON objects do not match, original: %s, reconstructed: %s", string(data), string(reconstructedData))
		}
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

// SerializeCosmosTx serializes a slice of sdk.Msg's using the CosmosTx type. The sdk.Msg's are
// packed into Any's and inserted into the Messages field of a CosmosTx. The CosmosTx is marshaled
// depending on the encoding type passed in. The marshaled bytes are returned.
func SerializeCosmosTx(cdc codec.BinaryCodec, msgs []proto.Message) ([]byte, error) {
	var (
		bz  []byte
		err error
	)
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
		return nil, errorsmod.Wrapf(err, "cannot marshal CosmosTx with protobuf")
	}

	return bz, nil
}

func equalJSON(a, b []byte) (bool, error) {
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &y); err != nil {
		return false, err
	}
	return reflect.DeepEqual(x, y), nil
}
