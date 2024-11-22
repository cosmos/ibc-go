package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
)

func TestABIFungibleTokenPacketData(t *testing.T) {
	data := types.FungibleTokenPacketData{
		Denom:    "denom",
		Sender:   "sender",
		Receiver: "receiver",
		Amount:   "100",
		Memo:     "memo",
	}

	encodedData, err := types.EncodeABIFungibleTokenPacketData(data)
	require.NoError(t, err)

	decodedData, err := types.DecodeABIFungibleTokenPacketData(encodedData)
	require.NoError(t, err)
	require.Equal(t, data, *decodedData)
}
