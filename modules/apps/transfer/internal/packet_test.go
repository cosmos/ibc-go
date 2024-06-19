package internal

import (
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
)

func TestUnmarshalPacketData(t *testing.T) {
	var (
		packetDataBz []byte
		version      string
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: v1 -> v2",
			func() {},
			nil,
		},
		{
			"success: v2",
			func() {
				packetData := types.NewFungibleTokenPacketDataV2(
					[]types.Token{
						{
							Denom:  types.NewDenom("atom", types.NewTrace("transfer", "channel-0")),
							Amount: "1000",
						},
					}, "sender", "receiver", "")

				packetDataBz = packetData.GetBytes()
				version = types.V2
			},
			nil,
		},
		{
			"invalid version",
			func() {
				version = "ics20-100"
			},
			types.ErrInvalidVersion,
		},
	}

	for _, tc := range testCases {

		packetDataV1 := types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", "sender", "receiver", "")

		packetDataBz = packetDataV1.GetBytes()
		version = types.V1

		tc.malleate()

		packetData, err := UnmarshalPacketData(packetDataBz, version)

		expPass := tc.expError == nil
		if expPass {
			require.IsType(t, types.FungibleTokenPacketDataV2{}, packetData)
		} else {
			require.ErrorIs(t, err, tc.expError)
		}
	}
}

func TestPacketV1ToPacketV2(t *testing.T) {
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
						Denom:  types.NewDenom("atom", types.NewTrace("transfer", "channel-0")),
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
						Denom:  types.NewDenom("atom"),
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
						Denom:  types.NewDenom("atom/withslash", types.NewTrace("transfer", "channel-0")),
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
						Denom:  types.NewDenom("atom/", types.NewTrace("transfer", "channel-0")),
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
						Denom:  types.NewDenom("atom/pool", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
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
						Denom:  types.NewDenom("atom", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1"), types.NewTrace("transfer-custom", "channel-2")),
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
						Denom:  types.NewDenom("atom/pool", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1"), types.NewTrace("transfer-custom", "channel-2")),
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
		actualV2Data, err := packetDataV1ToV2(tc.v1Data)

		expPass := tc.expError == nil
		if expPass {
			require.NoError(t, err, "test case: %s", tc.name)
			require.Equal(t, tc.v2Data, actualV2Data, "test case: %s", tc.name)
		} else {
			require.Error(t, err, "test case: %s", tc.name)
		}
	}
}
