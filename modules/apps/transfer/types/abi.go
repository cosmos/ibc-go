package types

import (
	"math/big"
	"reflect"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

type abiFungibleTokenPacketData struct {
	Denom    string
	Amount   *big.Int
	Sender   string
	Receiver string
	Memo     string
}

func EncodeABIFungibleTokenPacketData(data FungibleTokenPacketData) ([]byte, error) {
	amount, ok := sdkmath.NewIntFromString(data.Amount)
	if !ok {
		return nil, ErrInvalidAmount.Wrapf("unable to parse transfer amount (%s) into math.Int", data.Amount)
	}
	parsedABI, err := getFungibleTokenPacketDataABI()
	if err != nil {
		return nil, err
	}

	abiData := abiFungibleTokenPacketData{
		Denom:    data.Denom,
		Amount:   amount.BigInt(),
		Sender:   data.Sender,
		Receiver: data.Receiver,
		Memo:     data.Memo,
	}

	return parsedABI.Pack(abiData)
}

func DecodeABIFungibleTokenPacketData(abiEncodedData []byte) (*FungibleTokenPacketData, error) {
	parsedABI, err := getFungibleTokenPacketDataABI()
	if err != nil {
		return nil, err
	}

	// Unpack the data
	unpacked, err := parsedABI.Unpack(abiEncodedData)
	if err != nil {
		return nil, err
	}
	unpackedData := reflect.ValueOf(unpacked[0])

	amount := unpackedData.FieldByName("Amount").Interface().(*big.Int)
	amountStr := sdkmath.NewIntFromBigInt(amount).String()

	data := &FungibleTokenPacketData{
		Denom:    unpackedData.FieldByName("Denom").String(),
		Amount:   amountStr,
		Sender:   unpackedData.FieldByName("Sender").String(),
		Receiver: unpackedData.FieldByName("Receiver").String(),
		Memo:     unpackedData.FieldByName("Memo").String(),
	}

	return data, nil
}

func getFungibleTokenPacketDataABI() (abi.Arguments, error) {
	abiType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "Denom", Type: "string"},
		{Name: "Amount", Type: "uint256"},
		{Name: "Sender", Type: "string"},
		{Name: "Receiver", Type: "string"},
		{Name: "Memo", Type: "string"},
	})
	if err != nil {
		return abi.Arguments{}, err
	}

	return abi.Arguments{
		{
			Type: abiType,
		},
	}, nil
}
