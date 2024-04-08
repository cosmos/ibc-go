package convert

import (
	v1types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	v3types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types/v3"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConvertPacketV1ToPacketV2(t *testing.T) {
	const (
		sender   = "sender"
		receiver = "receiver"
	)

	testCases := []struct {
		name        string
		v1Data      v1types.FungibleTokenPacketData
		v2Data      v3types.FungibleTokenPacketData
		shouldPanic bool
	}{
		{
			"success",
			v1types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
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
			v1types.NewFungibleTokenPacketData("transfer/channel-0/atom/withslash", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
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
			v1types.NewFungibleTokenPacketData("transfer/channel-0/atom/", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
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
			v1types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/atom/pool", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
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
			v1types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom",
						Amount: 1000,
						Trace:  []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
					},
				}, sender, receiver, ""),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		shouldPanic := tc.shouldPanic
		if !shouldPanic {
			v2Data := PacketDataV1ToV3(tc.v1Data)
			require.Equal(t, tc.v2Data, v2Data)
		} else {
			require.Panicsf(t, func() {
				PacketDataV1ToV3(tc.v1Data)
			}, tc.name)
		}
	}
}
