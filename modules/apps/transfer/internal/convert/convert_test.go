package convert

import (
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func TestConvertPacketV1ToPacketV2(t *testing.T) {
	const (
		sender   = "sender"
		receiver = "receiver"
	)

	testCases := []struct {
		name     string
		v1Data   types.FungibleTokenPacketData
		v2Data   types.FungibleTokenPacketDataV2
		expError error
	}{
		{
			"success",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  "atom",
							Trace: []string{"transfer/channel-0"},
						},
						Amount: "1000",
					},
				}, sender, receiver, ""),
			nil,
		},
		{
			"success with empty trace",
			types.NewFungibleTokenPacketData("atom", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  "atom",
							Trace: nil,
						},
						Amount: "1000",
					},
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/withslash", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  "atom/withslash",
							Trace: []string{"transfer/channel-0"},
						},
						Amount: "1000",
					},
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: base denom with '/' at the end",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  "atom/",
							Trace: []string{"transfer/channel-0"},
						},
						Amount: "1000",
					},
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: longer trace base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/atom/pool", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  "atom/pool",
							Trace: []string{"transfer/channel-0", "transfer/channel-1"},
						},
						Amount: "1000",
					},
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: longer trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  "atom",
							Trace: []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
						},
						Amount: "1000",
					},
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: base denom with slash, trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom/pool", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  "atom/pool",
							Trace: []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
						},
						Amount: "1000",
					},
				}, sender, receiver, ""),
			nil,
		},
		{
			"failure: packet data fails validation with empty denom",
			types.NewFungibleTokenPacketData("", "1000", sender, receiver, ""),
			types.FungibleTokenPacketDataV2{},
			errorsmod.Wrap(types.ErrInvalidDenomForTransfer, "base denomination cannot be blank"),
		},
	}

	for _, tc := range testCases {
		actualV2Data, err := PacketDataV1ToV2(tc.v1Data)

		expPass := tc.expError == nil
		if expPass {
			require.NoError(t, err, "test case: %s", tc.name)
			require.Equal(t, tc.v2Data, actualV2Data, "test case: %s", tc.name)
		} else {
			require.Error(t, err, "test case: %s", tc.name)
		}
	}
}
