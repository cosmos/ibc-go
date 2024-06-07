package types_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
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
		expPass    bool
	}{
		{"valid packet", types.NewFungibleTokenPacketData(denom, amount, sender, receiver, ""), true},
		{"valid packet with memo", types.NewFungibleTokenPacketData(denom, amount, sender, receiver, "memo"), true},
		{"valid packet with large amount", types.NewFungibleTokenPacketData(denom, largeAmount, sender, receiver, ""), true},
		{"invalid denom", types.NewFungibleTokenPacketData("", amount, sender, receiver, ""), false},
		{"invalid denom, invalid portID", types.NewFungibleTokenPacketData("(tranfer)/channel-1/uatom", amount, sender, receiver, ""), false},
		{"invalid empty amount", types.NewFungibleTokenPacketData(denom, "", sender, receiver, ""), false},
		{"invalid zero amount", types.NewFungibleTokenPacketData(denom, "0", sender, receiver, ""), false},
		{"invalid negative amount", types.NewFungibleTokenPacketData(denom, "-1", sender, receiver, ""), false},
		{"invalid large amount", types.NewFungibleTokenPacketData(denom, invalidLargeAmount, sender, receiver, ""), false},
		{"missing sender address", types.NewFungibleTokenPacketData(denom, amount, emptyAddr, receiver, ""), false},
		{"missing recipient address", types.NewFungibleTokenPacketData(denom, amount, sender, emptyAddr, ""), false},
	}

	for i, tc := range testCases {
		tc := tc

		err := tc.packetData.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %v", i, err)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
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

// TestFungibleTokenPacketDataValidateBasic tests ValidateBasic for FungibleTokenPacketData
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
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				nil,
			),
			nil,
		},
		{
			"success: valid packet with memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"memo",
				nil,
			),
			nil,
		},
		{
			"success: valid packet with large amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: largeAmount,
					},
				},
				sender,
				receiver,
				"memo",
				nil,
			),
			nil,
		},
		{
			"success: valid packet with forwarding path hops",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  denom,
							Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")},
						},
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				&types.ForwardingInfo{
					Hops: []*types.Hop{
						{
							PortId:    "transfer",
							ChannelId: "channel-1",
						},
					},
					Memo: "",
				},
			),
			nil,
		},
		{
			"failure: invalid denom",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom("", types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				nil,
			),
			types.ErrInvalidDenomForTransfer,
		},
		{
			"failure: invalid empty amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: "",
					},
				},
				sender,
				receiver,
				"",
				nil,
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
				nil,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid zero amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: "0",
					},
				},
				sender,
				receiver,
				"",
				nil,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid negative amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: "-100",
					},
				},
				sender,
				receiver,
				"",
				nil,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: invalid large amount",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: invalidLargeAmount,
					},
				},
				sender,
				receiver,
				"memo",
				nil,
			),
			types.ErrInvalidAmount,
		},
		{
			"failure: missing sender address",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				"",
				receiver,
				"memo",
				nil,
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: missing recipient address",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				"",
				"",
				nil,
			),
			ibcerrors.ErrInvalidAddress,
		},
		{
			"failure: memo field too large",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: largeAmount,
					},
				},
				sender,
				receiver,
				ibctesting.GenerateString(types.MaximumMemoLength+1),
				nil,
			),
			types.ErrInvalidMemo,
		},
		{
			"failure: memo must be empty if forwarding path hops is not empty",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom: types.Denom{
							Base:  denom,
							Trace: []types.Trace{types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")},
						},
						Amount: amount,
					},
				},
				sender,
				receiver,
				"memo",
				&types.ForwardingInfo{
					Hops: []*types.Hop{
						{
							PortId:    "transfer",
							ChannelId: "channel-1",
						},
					},
					Memo: "",
				},
			),
			types.ErrInvalidMemo,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.packetData.ValidateBasic()

			expPass := tc.expErr == nil
			if expPass {
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
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				nil,
			),
			sender,
		},
		{
			"empty sender field",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				"",
				receiver,
				"abc",
				nil,
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
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, receiver),
				nil,
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
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, receiver),
				nil,
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
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				`{"src_callback": "string"}`,
				nil,
			),
			"string",
		},
		{
			"failure: src_callback key not found memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				fmt.Sprintf(`{"dest_callback": {"address": "%s", "min_gas": "200000"}}`, receiver),
				nil,
			),
			nil,
		},
		{
			"failure: empty memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				nil,
			),
			nil,
		},
		{
			"failure: non-json memo",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"invalid",
				nil,
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
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"",
				nil,
			),
			false,
		},
		{
			"non-empty memo field, resulting marshalled bytes should contain the memo field",
			types.NewFungibleTokenPacketDataV2(
				[]types.Token{
					{
						Denom:  types.NewDenom(denom, types.NewTrace("transfer", "channel-0"), types.NewTrace("transfer", "channel-1")),
						Amount: amount,
					},
				},
				sender,
				receiver,
				"abc",
				nil,
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
