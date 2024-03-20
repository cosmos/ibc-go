package transfer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func TestConvertPacketV1ToPacketV2(t *testing.T) {
	const (
		sender   = "sender"
		receiver = "receiver"
	)

	testCases := []struct {
		name        string
		v1Data      types.FungibleTokenPacketData
		v2Data      types.FungibleTokenPacketDataV2
		shouldPanic bool
	}{
		{
			"success",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]*types.Token{
					{
						Denom:  "atom",
						Amount: 1000,
						Trace:  []string{"transfer/channel-0"},
					},
				}, sender, receiver, ""),
			false,
		},
		{
			"success: base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/withslash", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]*types.Token{
					{
						Denom:  "atom/withslash",
						Amount: 1000,
						Trace:  []string{"transfer/channel-0"},
					},
				}, sender, receiver, ""),
			false,
		},
		{
			"success: base denom with '/' at the end",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]*types.Token{
					{
						Denom:  "atom/",
						Amount: 1000,
						Trace:  []string{"transfer/channel-0"},
					},
				}, sender, receiver, ""),
			false,
		},
		{
			"success: longer trace base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/atom/pool", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]*types.Token{
					{
						Denom:  "atom/pool",
						Amount: 1000,
						Trace:  []string{"transfer/channel-0", "transfer/channel-1"},
					},
				}, sender, receiver, ""),
			false,
		},
		{
			"success: longer trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]*types.Token{
					{
						Denom:  "atom",
						Amount: 1000,
						Trace:  []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
					},
				}, sender, receiver, ""),
			false,
		},

		// TODO: re-enable this test.
		// {"failure: empty path",
		//	types.NewFungibleTokenPacketData("", "1000", sender, receiver, ""),
		//	types.FungibleTokenPacketDataV2{},
		//	true,
		// },
	}

	for _, tc := range testCases {
		tc := tc

		shouldPanic := tc.shouldPanic
		if !shouldPanic {
			v2Data := ConvertPacketV1ToPacketV2(tc.v1Data)
			require.Equal(t, tc.v2Data, v2Data)
		} else {
			require.Panicsf(t, func() {
				ConvertPacketV1ToPacketV2(tc.v1Data)
			}, tc.name)
		}
	}
}
