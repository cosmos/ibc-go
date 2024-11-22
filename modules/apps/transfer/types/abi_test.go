package types_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
)

const solidityEncodedHex = "000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000f4240000000000000000000000000000000000000000000000000000000000000016000000000000000000000000000000000000000000000000000000000000000057561746f6d000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000673656e64657200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008726563656976657200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000046d656d6f00000000000000000000000000000000000000000000000000000000"

func TestEncodeABIFungibleTokenPacketData(t *testing.T) {
	data := types.FungibleTokenPacketData{
		Denom:    "uatom",
		Sender:   "sender",
		Receiver: "receiver",
		Amount:   "1000000",
		Memo:     "memo",
	}

	encodedData, err := types.EncodeABIFungibleTokenPacketData(data)
	require.NoError(t, err)

	hexEncodedData := hex.EncodeToString(encodedData)
	require.Equal(t, solidityEncodedHex, hexEncodedData)

	decodedData, err := types.DecodeABIFungibleTokenPacketData(encodedData)
	require.NoError(t, err)
	require.Equal(t, data, *decodedData)
}

func TestDecodeABIFungibleTokenPacketData(t *testing.T) {
	encodedData, err := hex.DecodeString(solidityEncodedHex)
	require.NoError(t, err)

	data, err := types.DecodeABIFungibleTokenPacketData(encodedData)
	require.NoError(t, err)

	expectedData := types.FungibleTokenPacketData{
		Denom:    "uatom",
		Sender:   "sender",
		Receiver: "receiver",
		Amount:   "1000000",
		Memo:     "memo",
	}

	require.Equal(t, expectedData, *data)
}
