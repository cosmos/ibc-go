package convert

import (
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	v3types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types/v3"
)

func TestConvertPacketV1ToPacketV3(t *testing.T) {
	const (
		sender   = "sender"
		receiver = "receiver"
	)

	testCases := []struct {
		name     string
		v1Data   types.FungibleTokenPacketData
		v3Data   v3types.FungibleTokenPacketData
		expPanic error
	}{
		{
			"success",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0"},
					},
				}, sender, receiver, "", nil),
			nil,
		},
		{
			"success with empty trace",
			types.NewFungibleTokenPacketData("atom", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom",
						Amount: "1000",
						Trace:  nil,
					},
				}, sender, receiver, "", nil),
			nil,
		},
		{
			"success: base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/withslash", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom/withslash",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0"},
					},
				}, sender, receiver, "", nil),
			nil,
		},
		{
			"success: base denom with '/' at the end",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom/",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0"},
					},
				}, sender, receiver, "", nil),
			nil,
		},
		{
			"success: longer trace base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/atom/pool", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				}, sender, receiver, "", nil),
			nil,
		},
		{
			"success: longer trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
					},
				}, sender, receiver, "", nil),
			nil,
		},
		{
			"success: base denom with slash, trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom/pool", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
					},
				}, sender, receiver, "", nil),
			nil,
		},
		{
			"failure: panics with empty denom",
			types.NewFungibleTokenPacketData("", "1000", sender, receiver, ""),
			v3types.FungibleTokenPacketData{},
			errorsmod.Wrap(types.ErrInvalidDenomForTransfer, "base denomination cannot be blank"),
		},
	}

	for _, tc := range testCases {
		expPass := tc.expPanic == nil
		if expPass {
			v3Data := PacketDataV1ToV3(tc.v1Data)
			require.Equal(t, tc.v3Data, v3Data, "test case: %s", tc.name)
		} else {
			require.PanicsWithError(t, tc.expPanic.Error(), func() {
				PacketDataV1ToV3(tc.v1Data)
			}, "test case: %s", tc.name)
		}
	}
}
