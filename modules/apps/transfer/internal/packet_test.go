package internal

import (
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
)

const (
	sender   = "sender"
	receiver = "receiver"
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
							Denom:  types.NewDenom("atom", types.NewHop("transfer", "channel-0")),
							Amount: "1000",
						},
					}, sender, receiver, "", types.ForwardingPacketData{})

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

		packetDataV1 := types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", sender, receiver, "")

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

// TestV2ForwardsCompatibilityFails asserts that new fields being added to a future proto definition of
// FungibleTokenPacketDataV2 fail to unmarshal with previous versions. In essence, permit backwards compatibility
// but restrict forward one.
func TestV2ForwardsCompatibilityFails(t *testing.T) {
	var (
		packet       types.FungibleTokenPacketDataV2
		packetDataBz []byte
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"failure: new field present in packet data",
			func() {
				// packet data containing extra field unknown to current proto file.
				packetDataBz = append(packet.GetBytes(), []byte("22\tnew_value")...)
			},
			ibcerrors.ErrInvalidType,
		},
	}

	for _, tc := range testCases {
		packet = types.NewFungibleTokenPacketDataV2(
			[]types.Token{
				{
					Denom:  types.NewDenom("atom", types.NewHop("transfer", "channel-0")),
					Amount: "1000",
				},
			}, "sender", "receiver", "", types.ForwardingPacketData{},
		)

		packetDataBz = packet.GetBytes()

		tc.malleate()

		packetData, err := UnmarshalPacketData(packetDataBz, types.V2)

		expPass := tc.expError == nil
		if expPass {
			require.NoError(t, err)
			require.NotEqual(t, types.FungibleTokenPacketDataV2{}, packetData)
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
						Denom:  types.NewDenom("atom", types.NewHop("transfer", "channel-0")),
						Amount: "1000",
					},
				}, sender, receiver, "", types.ForwardingPacketData{}),
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
				}, sender, receiver, "", types.ForwardingPacketData{}),
			nil,
		},
		{
			"success: base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/withslash", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom("atom/withslash", types.NewHop("transfer", "channel-0")),
						Amount: "1000",
					},
				}, sender, receiver, "", types.ForwardingPacketData{}),
			nil,
		},
		{
			"success: base denom with '/' at the end",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom("atom/", types.NewHop("transfer", "channel-0")),
						Amount: "1000",
					},
				}, sender, receiver, "", types.ForwardingPacketData{}),
			nil,
		},
		{
			"success: longer trace base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/atom/pool", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom("atom/pool", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: "1000",
					},
				}, sender, receiver, "", types.ForwardingPacketData{}),
			nil,
		},
		{
			"success: longer trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom("atom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1"), types.NewHop("transfer-custom", "channel-2")),
						Amount: "1000",
					},
				}, sender, receiver, "", types.ForwardingPacketData{}),
			nil,
		},
		{
			"success: base denom with slash, trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom/pool", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom("atom/pool", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1"), types.NewHop("transfer-custom", "channel-2")),
						Amount: "1000",
					},
				}, sender, receiver, "", types.ForwardingPacketData{}),
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
