package types_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/gogoproto/proto"
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
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				"",
			),
			nil,
		},
		{
			"success: valid packet with memo",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				"memo",
			),
			nil,
		},
		{
			"success: valid packet with large amount",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: largeAmount,
				},
				sender,
				receiver,
				"memo",
			),
			nil,
		},
		{
			"failure: invalid denom",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom("", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid empty amount",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: "",
				},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid zero amount",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: "0",
				},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid negative amount",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: "-100",
				},
				sender,
				receiver,
				"",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid large amount",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: invalidLargeAmount,
				},
				sender,
				receiver,
				"memo",
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: missing sender address",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				"",
				receiver,
				"memo",
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: missing recipient address",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				"",
				"",
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: memo field too large",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: largeAmount,
				},
				sender,
				receiver,
				ibctesting.GenerateString(types.MaximumMemoLength+1),
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
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				"",
			),
			sender,
		},
		{
			"empty sender field",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				"",
				receiver,
				"abc",
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
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, receiver),
			),

			map[string]interface{}{
				"address": receiver,
			},
		},
		{
			"success: src_callback key in memo with additional fields",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, receiver),
			),
			map[string]interface{}{
				"address":   receiver,
				"gas_limit": "200000",
			},
		},
		{
			"success: src_callback has string value",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				`{"src_callback": "string"}`,
			),
			"string",
		},
		{
			"failure: src_callback key not found memo",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				fmt.Sprintf(`{"dest_callback": {"address": "%s", "min_gas": "200000"}}`, receiver),
			),
			nil,
		},
		{
			"failure: empty memo",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				"",
			),
			nil,
		},
		{
			"failure: non-json memo",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				"invalid",
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
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				"",
			),
			false,
		},
		{
			"non-empty memo field, resulting marshalled bytes should contain the memo field",
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom(denom, types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: amount,
				},
				sender,
				receiver,
				"abc",
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
		encoding     string
	)

	testCases := []struct {
		name     string
		malleate func()
		expError error
	}{
		{
			"success: v1 -> v2 with empty encoding (JSON)",
			func() {},
			nil,
		},
		{
			"success: v1 -> v2 with JSON encoding",
			func() {
				encoding = types.EncodingJSON
			},
			nil,
		},
		{
			"success: v1 -> v2 with protobuf encoding",
			func() {
				packetData := types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", sender, receiver, "")
				bz, err := proto.Marshal(&packetData)
				require.NoError(t, err)

				packetDataBz = bz
				encoding = types.EncodingProtobuf
			},
			nil,
		},
		{
			"success: v1 -> v2 with abi encoding",
			func() {
				packetData := types.NewFungibleTokenPacketData("transfer/channel-0/atom", "1000", sender, receiver, "")
				bz, err := types.EncodeABIFungibleTokenPacketData(&packetData)
				require.NoError(t, err)

				packetDataBz = bz
				encoding = types.EncodingABI
			},
			nil,
		},
		{
			"failure: v2 with empty encoding (protobuf)",
			func() {
				packetData := types.NewFungibleTokenPacketDataV2(
					types.Token{
						Denom:  types.NewDenom("atom", types.NewHop("transfer", "channel-0")),
						Amount: "1000",
					}, sender, receiver, "")

				packetDataBz = packetData.GetBytes()
				version = "ics20-2"
			},
			types.ErrInvalidVersion,
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
		encoding = ""

		tc.malleate()

		packetData, err := types.UnmarshalPacketData(packetDataBz, version, encoding)

		if tc.expError == nil {
			require.NoError(t, err)
			require.NotEmpty(t, packetData.Token)
			require.NotEmpty(t, packetData.Sender)
			require.NotEmpty(t, packetData.Receiver)
			require.IsType(t, types.FungibleTokenPacketDataV2{}, packetData)
		} else {
			ibctesting.RequireErrorIsOrContains(t, err, tc.expError)
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
				types.Token{
					Denom:  types.NewDenom("atom", types.NewHop("transfer", "channel-0")),
					Amount: "1000",
				}, sender, receiver, ""),
			nil,
		},
		{
			"success with empty trace",
			types.NewFungibleTokenPacketData("atom", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom("atom"),
					Amount: "1000",
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/withslash", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom("atom/withslash", types.NewHop("transfer", "channel-0")),
					Amount: "1000",
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: base denom with '/' at the end",
			types.NewFungibleTokenPacketData("transfer/channel-0/atom/", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom("atom/", types.NewHop("transfer", "channel-0")),
					Amount: "1000",
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: longer trace base denom with '/'",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/atom/pool", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom("atom/pool", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1")),
					Amount: "1000",
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: longer trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom("atom", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1"), types.NewHop("transfer-custom", "channel-2")),
					Amount: "1000",
				}, sender, receiver, ""),
			nil,
		},
		{
			"success: base denom with slash, trace with non transfer port",
			types.NewFungibleTokenPacketData("transfer/channel-0/transfer/channel-1/transfer-custom/channel-2/atom/pool", "1000", sender, receiver, ""),
			types.NewFungibleTokenPacketDataV2(
				types.Token{
					Denom:  types.NewDenom("atom/pool", types.NewHop("transfer", "channel-0"), types.NewHop("transfer", "channel-1"), types.NewHop("transfer-custom", "channel-2")),
					Amount: "1000",
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
		actualV2Data, err := types.PacketDataV1ToV2(tc.v1Data)

		if tc.expError == nil {
			require.NoError(t, err, "test case: %s", tc.name)
			require.Equal(t, tc.v2Data, actualV2Data, "test case: %s", tc.name)
		} else {
			require.Error(t, err, "test case: %s", tc.name)
		}
	}
}
