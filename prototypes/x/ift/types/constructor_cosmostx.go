package types

import (
	"strings"

	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/bech32"

	gmptypes "github.com/cosmos/ibc-go/v11/modules/apps/27-gmp/types"
)

// CosmosTxConstructor handles mint call construction for Cosmos SDK chains
type CosmosTxConstructor struct{}

func (CosmosTxConstructor) ValidateCounterpartyAddress(address string) error {
	// Cosmos addresses are validated as bech32, but the counterparty IFT address
	// is typically the module account which we cannot validate format-wise
	// since bech32 prefix varies by chain
	if strings.TrimSpace(address) == "" {
		return ErrInvalidReceiver
	}
	return nil
}

// validateBech32Address validates that an address is valid bech32 format
// with correct data length, without requiring a specific prefix
func validateBech32Address(address string) error {
	if strings.TrimSpace(address) == "" {
		return ErrInvalidReceiver.Wrap("address cannot be empty")
	}

	_, data, err := bech32.DecodeAndConvert(address)
	if err != nil {
		return ErrInvalidReceiver.Wrapf("invalid bech32: %s", err)
	}

	// Standard Cosmos addresses are 20 bytes
	const addrLen = 20
	if len(data) != addrLen {
		return ErrInvalidReceiver.Wrapf("invalid address length: expected %d bytes, got %d", addrLen, len(data))
	}

	return nil
}

func (CosmosTxConstructor) ConstructMintCall(cdc codec.BinaryCodec, receiver string, amount math.Int, denom string, counterpartyIcaAddress string) ([]byte, error) {
	if err := validateBech32Address(receiver); err != nil {
		return nil, err
	}

	msg := &MsgIFTMint{
		Signer:   counterpartyIcaAddress,
		Denom:    denom,
		Receiver: receiver,
		Amount:   amount,
	}

	return gmptypes.SerializeCosmosTx(cdc, []proto.Message{msg})
}
