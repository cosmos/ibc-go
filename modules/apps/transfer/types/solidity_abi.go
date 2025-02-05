package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"

	errorsmod "cosmossdk.io/errors"
)

// getICS20ABI returns an abi.Arguments slice describing the Solidity types of the struct.
func getICS20ABI() abi.Arguments {
	// Create the ABI types for each field.
	// The Solidity types used are:
	// - string for Denom, Sender, Receiver and Memo.
	// - uint256 for Amount.
	tupleType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{
			Name: "denom",
			Type: "string",
		},
		{
			Name: "sender",
			Type: "string",
		},
		{
			Name: "receiver",
			Type: "string",
		},
		{
			Name: "amount",
			Type: "uint256",
		},
		{
			Name: "memo",
			Type: "string",
		},
	})
	if err != nil {
		panic(err)
	}

	// Create an ABI argument representing our struct as a single tuple argument.
	arguments := abi.Arguments{
		{
			Type: tupleType,
		},
	}

	return arguments
}

// DecodeABIFungibleTokenPacketData decodes a solidity ABI encoded ics20lib.ICS20LibFungibleTokenPacketData
// and converts it into an ibc-go FungibleTokenPacketData.
func DecodeABIFungibleTokenPacketData(data []byte) (*FungibleTokenPacketData, error) {
	arguments := getICS20ABI()

	packetDataI, err := arguments.Unpack(data)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrAbiDecoding, "failed to unpack data: %s", err)
	}

	packetData, ok := packetDataI[0].(struct {
		Denom    string   `json:"denom"`
		Sender   string   `json:"sender"`
		Receiver string   `json:"receiver"`
		Amount   *big.Int `json:"amount"`
		Memo     string   `json:"memo"`
	})
	if !ok {
		return nil, errorsmod.Wrapf(ErrAbiDecoding, "failed to parse packet data")
	}

	return &FungibleTokenPacketData{
		Denom:    packetData.Denom,
		Sender:   packetData.Sender,
		Receiver: packetData.Receiver,
		Amount:   packetData.Amount.String(),
		Memo:     packetData.Memo,
	}, nil
}

func EncodeABIFungibleTokenPacketData(data *FungibleTokenPacketData) ([]byte, error) {
	amount, ok := new(big.Int).SetString(data.Amount, 10)
	if !ok {
		return nil, errorsmod.Wrapf(ErrAbiEncoding, "failed to parse amount: %s", data.Amount)
	}

	packetData := struct {
		Denom    string   `json:"denom"`
		Sender   string   `json:"sender"`
		Receiver string   `json:"receiver"`
		Amount   *big.Int `json:"amount"`
		Memo     string   `json:"memo"`
	}{
		data.Denom,
		data.Sender,
		data.Receiver,
		amount,
		data.Memo,
	}

	arguments := getICS20ABI()
	// Pack the values in the order defined in the ABI.
	encodedData, err := arguments.Pack(packetData)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrAbiEncoding, "failed to pack data: %s", err)
	}

	return encodedData, nil
}
