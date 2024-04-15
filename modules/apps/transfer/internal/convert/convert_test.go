package convert

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	v3types "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types/v3"
)

func TestConvertICS20V1ToV2(t *testing.T) {
	const (
		sender   = "sender"
		receiver = "receiver"
	)

	testCases := []struct {
		name        string
		v1Data      v1types.FungibleTokenPacketData
		v3Data      v3types.FungibleTokenPacketData
		shouldPanic bool
	}{
		{
			"success",
			v1types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom",
						Amount: "1000",
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
						Amount: "1000",
						Trace:  []string{"transfer/channel-0"},
					},
				}, sender, receiver, ""),
			false,
		},
		// TODO: this test should pass, but v1 packet data validation is failing with this denom.
		// https://github.com/cosmos/ibc-go/issues/6124
		// {
		//	"success: base denom with '/' at the end",
		//	v1types.NewFungibleTokenPacketData("transfer/channel-0/atom/", "1000", sender, receiver, ""),
		//	v3types.NewFungibleTokenPacketData(
		//		[]*v3types.Token{
		//			{
		//				Denom:  "atom/",
		//				Amount: "1000",
		//				Trace:  []string{"transfer/channel-0"},
		//			},
		//		}, sender, receiver, ""),
		//	false,
		// },
		{
			"success: longer trace base denom with '/'",
			v1types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/atom/pool", "1000", sender, receiver, ""),
			v3types.NewFungibleTokenPacketData(
				[]*v3types.Token{
					{
						Denom:  "atom/pool",
						Amount: "1000",
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
						Amount: "1000",
						Trace:  []string{"transfer/channel-0", "transfer/channel-1", "transfer-custom/channel-2"},
					},
				}, sender, receiver, ""),
			false,
		},
		{
			"failure: panics with empty denom",
			v1types.NewFungibleTokenPacketData("", "1000", sender, receiver, ""),
			v3types.FungibleTokenPacketData{
				Tokens: []*v3types.Token{
					{
						Denom:  "",
						Amount: "1000",
						Trace:  nil,
					},
				},
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		shouldPanic := tc.shouldPanic
		if !shouldPanic {
			v3Data := ICS20V1ToV2(tc.v1Data)
			require.Equal(t, tc.v3Data, v3Data)
		} else {
			require.Panicsf(t, func() {
				ICS20V1ToV2(tc.v1Data)
			}, tc.name)
		}
	}
}
