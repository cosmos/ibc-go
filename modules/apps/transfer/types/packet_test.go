package types_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
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
		{"invalid empty amount", types.NewFungibleTokenPacketData(denom, "", sender, receiver, ""), false},
		{"invalid zero amount", types.NewFungibleTokenPacketData(denom, "0", sender, receiver, ""), false},
		{"invalid negative amount", types.NewFungibleTokenPacketData(denom, "-1", sender, receiver, ""), false},
		{"invalid large amount", types.NewFungibleTokenPacketData(denom, invalidLargeAmount, sender, receiver, ""), false},
		{"missing sender address", types.NewFungibleTokenPacketData(denom, amount, emptyAddr, receiver, ""), false},
		{"missing recipient address", types.NewFungibleTokenPacketData(denom, amount, sender, emptyAddr, ""), false},
	}

	for i, tc := range testCases {
		err := tc.packetData.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %v", i, err)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

func (suite *TypesTestSuite) TestGetSourceCallbackAddress() {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expAddress string
	}{
		{
			"success: memo has callbacks in json struct and properly formatted src_callback_address which does not match packet sender",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, receiver),
			},
			receiver,
		},
		{
			"success: valid src_callback address specified in memo that matches sender",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, sender),
			},
			sender,
		},
		{
			"failure: memo is empty",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "",
			},
			"",
		},
		{
			"failure: memo is not json string",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "memo",
			},
			"",
		},
		{
			"failure: memo does not have callbacks in json struct",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"Key": 10}`,
			},
			"",
		},
		{
			"failure:  memo has src_callback in json struct but does not have address key",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"Key": 10}}`,
			},
			"",
		},
		{
			"failure: memo has src_callback in json struct but does not have string value for address key",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"address": 10}}`,
			},
			"",
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			srcCbAddr := tc.packetData.GetSourceCallbackAddress()
			suite.Require().Equal(tc.expAddress, srcCbAddr)
			suite.Require().Equal(sender, tc.packetData.GetPacketSender(""))
		})
	}
}

func (suite *TypesTestSuite) TestGetDestCallbackAddress() {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expAddress string
	}{
		{
			"success: memo has dest_callback in json struct and properly formatted dest_callback_address which does not match packet receiver",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, sender),
			},
			sender,
		},
		{
			"success: valid dest_callback address specified in memo that matches receiver",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, receiver),
			},
			receiver,
		},
		{
			"failure: memo is empty",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "",
			},
			"",
		},
		{
			"failure: memo is not json string",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "memo",
			},
			"",
		},
		{
			"failure: memo does not have callbacks in json struct",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"Key": 10}`,
			},
			"",
		},
		{
			"failure: memo has dest_callback in json struct but does not have address key",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"dest_callback": {"Key": 10}}`,
			},
			"",
		},
		{
			"failure: memo has dest_callback in json struct but does not have string value for address key",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"dest_callback": {"address": 10}}`,
			},
			"",
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			destCbAddr := tc.packetData.GetDestCallbackAddress()
			suite.Require().Equal(tc.expAddress, destCbAddr)
		})
	}
}

func (suite *TypesTestSuite) TestSourceUserDefinedGasLimit() {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expUserGas uint64
	}{
		{
			"success: memo is empty",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "",
			},
			0,
		},
		{
			"success: memo has user defined gas limit",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"gas_limit": "100"}}`,
			},
			100,
		},
		{
			"failure: memo has user defined gas limit as json number",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"gas_limit": 100}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as negative",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"gas_limit": "-100"}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as string",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"gas_limit": "invalid"}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as empty string",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"gas_limit": ""}}`,
			},
			0,
		},
		{
			"failure: malformed memo",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `invalid`,
			},
			0,
		},
	}

	for _, tc := range testCases {
		suite.Require().Equal(tc.expUserGas, tc.packetData.GetSourceUserDefinedGasLimit())
	}
}

func (suite *TypesTestSuite) TestDestUserDefinedGasLimit() {
	testCases := []struct {
		name       string
		packetData types.FungibleTokenPacketData
		expUserGas uint64
	}{
		{
			"success: memo is empty",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "",
			},
			0,
		},
		{
			"success: memo has user defined gas limit",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"dest_callback": {"gas_limit": "100"}}`,
			},
			100,
		},
		{
			"failure: memo has user defined gas limit as json number",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"dest_callback": {"gas_limit": 100}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as negative",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"dest_callback": {"gas_limit": "-100"}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as string",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"dest_callback": {"gas_limit": "invalid"}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as empty string",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"dest_callback": {"gas_limit": ""}}`,
			},
			0,
		},
		{
			"failure: malformed memo",
			types.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `invalid`,
			},
			0,
		},
	}

	for _, tc := range testCases {
		suite.Require().Equal(tc.expUserGas, tc.packetData.GetDestUserDefinedGasLimit())
	}
}
