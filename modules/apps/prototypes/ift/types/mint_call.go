package types

import (
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
)

// ConstructMintCall constructs the payload for minting tokens on the counterparty chain.
// Used for EVM and Cosmos constructors. Solana uses NewSolanaConstructor directly.
func ConstructMintCall(
	cdc codec.BinaryCodec,
	receiver string,
	amount math.Int,
	constructorType string,
	denom string,
	icaAddress string,
) ([]byte, error) {
	baseType := ParseConstructorType(constructorType)
	switch baseType {
	case ConstructorEVM:
		return EVMConstructor{}.ConstructMintCall(cdc, receiver, amount, denom, icaAddress)
	case ConstructorCosmos:
		return CosmosTxConstructor{}.ConstructMintCall(cdc, receiver, amount, denom, icaAddress)
	default:
		return nil, ErrInvalidConstructorType.Wrapf("type: %s", constructorType)
	}
}
