package types

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	errorsmod "cosmossdk.io/errors"
)

// getICS27ABI returns an abi.Arguments slice describing the Solidity types of the struct.
func getICS27ABI() abi.Arguments {
	// Create the ABI types for each field.
	// The Solidity types used are:
	// - string for Sender, Receiver and Memo.
	// - bytes for Salt and Payload.
	tupleType, err := abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{
			Name: "sender",
			Type: "string",
		},
		{
			Name: "receiver",
			Type: "string",
		},
		{
			Name: "salt",
			Type: "bytes",
		},
		{
			Name: "payload",
			Type: "bytes",
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

// DecodeABIGMPPacketData decodes a solidity ABI encoded ics27lib.GMPPacketData and converts it into an ibc-go GMPPacketData.
func DecodeABIGMPPacketData(data []byte) (*GMPPacketData, error) {
	arguments := getICS27ABI()

	packetDataI, err := arguments.Unpack(data)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrAbiDecoding, "failed to unpack data: %s", err)
	}

	packetData, ok := packetDataI[0].(struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
		Salt     []byte `json:"salt"`
		Payload  []byte `json:"payload"`
		Memo     string `json:"memo"`
	})
	if !ok {
		return nil, errorsmod.Wrapf(ErrAbiDecoding, "failed to parse packet data")
	}

	return &GMPPacketData{
		Sender:   packetData.Sender,
		Receiver: packetData.Receiver,
		Salt:     packetData.Salt,
		Payload:  packetData.Payload,
		Memo:     packetData.Memo,
	}, nil
}

// EncodeABIGMPPacketData encodes a GMPPacketData into a solidity ABI encoded byte array.
func EncodeABIGMPPacketData(data *GMPPacketData) ([]byte, error) {
	packetData := struct {
		Sender   string `json:"sender"`
		Receiver string `json:"receiver"`
		Salt     []byte `json:"salt"`
		Payload  []byte `json:"payload"`
		Memo     string `json:"memo"`
	}{
		data.Sender,
		data.Receiver,
		data.Salt,
		data.Payload,
		data.Memo,
	}

	arguments := getICS27ABI()
	// Pack the values in the order defined in the ABI.
	encodedData, err := arguments.Pack(packetData)
	if err != nil {
		return nil, errorsmod.Wrapf(ErrAbiEncoding, "failed to pack data: %s", err)
	}

	return encodedData, nil
}
