package types

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
)

// EVMConstructor handles mint call construction for EVM-based chains
type EVMConstructor struct{}

// iftMintSelector is the function selector for iftMint(address,uint256)
// keccak256("iftMint(address,uint256)")[:4] = 0x0a7244e7
var iftMintSelector = []byte{0x0a, 0x72, 0x44, 0xe7}

func (EVMConstructor) ValidateCounterpartyAddress(address string) error {
	if !common.IsHexAddress(address) {
		return ErrInvalidEVMAddress.Wrapf("address: %s", address)
	}
	return nil
}

func (EVMConstructor) validateReceiverAddress(address string) error {
	if !common.IsHexAddress(address) {
		return ErrInvalidEVMAddress.Wrapf("address: %s", address)
	}

	addr := common.HexToAddress(address)
	if addr == (common.Address{}) {
		return ErrZeroAddress
	}

	return nil
}

func (c EVMConstructor) ConstructMintCall(_ codec.BinaryCodec, receiver string, amount math.Int, _, _ string) ([]byte, error) {
	if err := c.validateReceiverAddress(receiver); err != nil {
		return nil, err
	}

	receiverAddr := common.HexToAddress(receiver)

	addressType, _ := abi.NewType("address", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)

	arguments := abi.Arguments{
		{Type: addressType},
		{Type: uint256Type},
	}

	packed, err := arguments.Pack(receiverAddr, amount.BigInt())
	if err != nil {
		return nil, ErrABIPackFailed.Wrap(err.Error())
	}

	return append(iftMintSelector, packed...), nil
}
