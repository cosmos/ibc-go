package types_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v9/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

const (
	denom              = "transfer/gaiachannel/atom"
	amount             = "100"
	largeAmount        = "18446744073709551616"                                                           // one greater than largest uint64 (^uint64(0))
	invalidLargeAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639936" // 2^256
)

// TestFungibleTokenPacketDataValidateBasic tests ValidateBasic for FungibleTokenPacketData
func TestFungibleTokenPacketDataValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expErr     error
	}{
		{"valid packet", types.NewFungibleTokenPacketData(denom, amount, sender, receiver, ""), nil},
		{"valid packet with memo", types.NewFungibleTokenPacketData(denom, amount, sender, receiver, "memo"), nil},
		{"valid packet with large amount", types.NewFungibleTokenPacketData(denom, largeAmount, sender, receiver, ""), nil},
		{"invalid denom", types.NewFungibleTokenPacketData("", amount, sender, receiver, ""), types.ErrInvalidDenomForTransfer},
		{"invalid denom, invalid portID", types.NewFungibleTokenPacketData("(transfer)/channel-1/uatom", amount, sender, receiver, ""), host.ErrInvalidID},
		{"invalid empty amount", types.NewFungibleTokenPacketData(denom, "", sender, receiver, ""), types.ErrInvalidAmount},
		{"invalid zero amount", types.NewFungibleTokenPacketData(denom, "0", sender, receiver, ""), types.ErrInvalidAmount},
		{"invalid negative amount", types.NewFungibleTokenPacketData(denom, "-1", sender, receiver, ""), types.ErrInvalidAmount},
		{"invalid large amount", types.NewFungibleTokenPacketData(denom, invalidLargeAmount, sender, receiver, ""), types.ErrInvalidAmount},
		{"missing sender address", types.NewFungibleTokenPacketData(denom, amount, emptyAddr, receiver, ""), ibcerrors.ErrInvalidAddress},
		{"missing recipient address", types.NewFungibleTokenPacketData(denom, amount, sender, emptyAddr, ""), ibcerrors.ErrInvalidAddress},
	}

	for i, tc := range testCases {
		tc := tc

		err := tc.packetData.ValidateBasic()
		if tc.expErr == nil {
			require.NoError(t, err, "valid test case %d failed: %v", i, err)
		} else {
			require.ErrorIs(t, err, tc.expErr, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func (suite *TypesTestSuite) TestGetPacketSender() {
	packetData := types.FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		Memo:     "",
	}

	suite.Require().Equal(sender, packetData.GetPacketSender(types.PortID))
}

func (suite *TypesTestSuite) TestPacketDataProvider() {
	testCases := []struct {
		name          string
		packetData    types.FungibleTokenPacketData
		expCustomData interface{}
	}{
		{
			"success: src_callback key in memo",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, receiver),
			},
			map[string]interface{}{
				"address": receiver,
			},
		},
		{
			"success: src_callback key in memo with additional fields",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, receiver),
			},
			map[string]interface{}{
				"address":   receiver,
				"gas_limit": "200000",
			},
		},
		{
			"success: src_callback has string value",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": "string"}`,
			},
			"string",
		},
		{
			"failure: empty memo",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "",
			},
			nil,
		},
		{
			"failure: non-json memo",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "invalid",
			},
			nil,
		},
	}

	for _, tc := range testCases {
		tc := tc

		customData := tc.packetData.GetCustomPacketData("src_callback")
		suite.Require().Equal(tc.expCustomData, customData)
	}
}

func (suite *TypesTestSuite) TestFungibleTokenPacketDataOmitEmpty() {
	// check that omitempty is present for the memo field
	packetData := types.FungibleTokenPacketData{
		Denom:    denom,
		Amount:   amount,
		Sender:   sender,
		Receiver: receiver,
		// Default value for non-specified memo field is empty string
	}

	bz, err := json.Marshal(packetData)
	suite.Require().NoError(err)

	// check that the memo field is not present in the marshalled bytes
	suite.Require().NotContains(string(bz), "memo")

	packetData.Memo = "abc"
	bz, err = json.Marshal(packetData)
	suite.Require().NoError(err)

	// check that the memo field is present in the marshalled bytes
	suite.Require().Contains(string(bz), "memo")
}

// TestFungibleTokenPacketDataV2ValidateBasic tests ValidateBasic for FungibleTokenPacketData
func TestFungibleTokenPacketDataV2ValidateBasic(t *testing.T) {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketDataV2
		expErr     error
	}{
		{
			"success: valid packet",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			nil,
		},
		{
			"success: valid packet with memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"memo",
				ibctesting.EmptyForwardingPacketData,
			),
			nil,
		},
		{
			"success: valid packet with large amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: largeAmount,
					},
				},
				sender,
				receiver,
				"memo",
				ibctesting.EmptyForwardingPacketData,
			),
			nil,
		},
		{
			"success: valid packet with forwarding path hops",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				types.NewForwardingPacketData("", validHop, validHop),
			),
			nil,
		},
		{
			"success: valid packet with forwarding path hops with memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				types.NewForwardingPacketData("memo", validHop),
			),
			nil,
		},
		{
			"failure: invalid denom",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom("", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			types.ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid empty amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: "",
					},
				},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid empty token array",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid zero amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: "0",
					},
				},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid negative amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: "-100",
					},
				},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid large amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: invalidLargeAmount,
					},
				},
				sender,
				receiver,
				"memo",
				ibctesting.EmptyForwardingPacketData,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: missing sender address",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				"",
				receiver,
				"memo",
				ibctesting.EmptyForwardingPacketData,
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: missing recipient address",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				"",
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: memo field too large",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: largeAmount,
					},
				},
				sender,
				receiver,
				ibctesting.GenerateString(types.MaximumMemoLength+1),
				ibctesting.EmptyForwardingPacketData,
			),
			types.ErrInvalidMemo,
		},
		{
			"failure: memo must be empty if forwarding path hops is not empty",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"memo",
				types.NewForwardingPacketData("", validHop),
			),
			types.ErrInvalidMemo,
		},
		{
			"failure: invalid forwarding path port ID",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				types.NewForwardingPacketData(
					"",
					types.NewHop(invalidPort, "channel-1"),
				),
			),
			types.ErrInvalidForwarding,
		},
		{
			"failure: invalid forwarding path channel ID",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				types.NewForwardingPacketData(
					"",
					types.NewHop("transfer", invalidChannel),
				),
			),
			types.ErrInvalidForwarding,
		},
		{
			"failure: invalid forwarding path too many hops",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				types.NewForwardingPacketData("", generateHops(types.MaximumNumberOfForwardingHops+1)...),
			),
			types.ErrInvalidForwarding,
		},
		{
			"failure: invalid forwarding path too long memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				types.NewForwardingPacketData(ibctesting.GenerateString(types.MaximumMemoLength+1), validHop),
			),
			types.ErrInvalidMemo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.packetData.ValidateBasic()

			if tc.expErr == nil {
				require.NoError(t, err, tc.name)
			} else {
				require.ErrorContains(t, err, tc.expErr.Error(), tc.name)
			}
		})
	}
}

func TestGetPacketSender(t *testing.T) {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketDataV2
		expSender  string
	}{
		{
			"non-empty sender field",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			sender,
		},
		{
			"empty sender field",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				"",
				receiver,
				"abc",
				ibctesting.EmptyForwardingPacketData,
			),
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expSender, tc.packetData.GetPacketSender(types.PortID))
		})
	}
}

func TestPacketDataProvider(t *testing.T) {
	testCases := []struct {
		name          string
		packetData    types.FungibleTokenPacketDataV2
		expCustomData interface{}
	}{
		{
			"success: src_callback key in memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, receiver),
				ibctesting.EmptyForwardingPacketData,
			),

			map[string]interface{}{
				"address": receiver,
			},
		},
		{
			"success: src_callback key in memo with additional fields",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, receiver),
				ibctesting.EmptyForwardingPacketData,
			),
			map[string]interface{}{
				"address":   receiver,
				"gas_limit": "200000",
			},
		},
		{
			"success: src_callback has string value",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				`{"src_callback": "string"}`,
				ibctesting.EmptyForwardingPacketData,
			),
			"string",
		},
		{
			"failure: src_callback key not found memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"dest_callback": {"address": "%s", "min_gas": "200000"}}`, receiver),
				ibctesting.EmptyForwardingPacketData,
			),
			nil,
		},
		{
			"failure: empty memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			nil,
		},
		{
			"failure: non-json memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"invalid",
				ibctesting.EmptyForwardingPacketData,
			),
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			customData := tc.packetData.GetCustomPacketData("src_callback")
			require.Equal(t, tc.expCustomData, customData)
		})
	}
}

func TestFungibleTokenPacketDataOmitEmpty(t *testing.T) {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketDataV2
		expMemo    bool
	}{
		{
			"empty memo field, resulting marshalled bytes should not contain the memo field",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				ibctesting.EmptyForwardingPacketData,
			),
			false,
		},
		{
			"non-empty memo field, resulting marshalled bytes should contain the memo field",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"abc",
				ibctesting.EmptyForwardingPacketData,
			),
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bz, err := json.Marshal(tc.packetData)
			if tc.expMemo {
				require.NoError(t, err, tc.name)
				// check that the memo field is present in the marshalled bytes
				require.Contains(t, string(bz), "memo")
			} else {
				require.NoError(t, err, tc.name)
				// check that the memo field is not present in the marshalled bytes
				require.NotContains(t, string(bz), "memo")
			}
		})
	}
}

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

		packetData, err := types.UnmarshalPacketData(packetDataBz, version)

		if tc.expError == nil {
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

		packetData, err := types.UnmarshalPacketData(packetDataBz, types.V2)

		if tc.expError == nil {
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
		actualV2Data, err := types.PacketDataV1ToV2(tc.v1Data)

		if tc.expError == nil {
			require.NoError(t, err, "test case: %s", tc.name)
			require.Equal(t, tc.v2Data, actualV2Data, "test case: %s", tc.name)
		} else {
			require.Error(t, err, "test case: %s", tc.name)
		}
	}
}
