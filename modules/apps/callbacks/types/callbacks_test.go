package types_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	transfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (suite *CallbacksTypesTestSuite) TestGetSourceCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	var packetData []byte
	testCases := []struct {
		name            string
		malleate        func()
		remainingGas    uint64
		expCallbackData types.CallbackData
		expPass         bool
	}{
		{
			"success: source callback",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     100000,
			},
			true,
		},
		{
			"success: source callback with gas limit",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "50000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     50000,
			},
			true,
		},
		{
			"success: source callback with too much gas limit",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     100000,
			},
			true,
		},
		{
			"failure: invalid packet data",
			func() {
				packetData = []byte("invalid packet data")
			},
			100000,
			types.CallbackData{},
			false,
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		packetUnmarshaler := transfer.IBCModule{}

		callbackData, err := types.GetSourceCallbackData(packetUnmarshaler, packetData, tc.remainingGas)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().Equal(tc.expCallbackData, callbackData)
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *CallbacksTypesTestSuite) TestGetDestCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	var packetData []byte
	testCases := []struct {
		name            string
		malleate        func()
		remainingGas    uint64
		expCallbackData types.CallbackData
		expPass         bool
	}{
		{
			"success: destination callback",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     100000,
			},
			true,
		},
		{
			"success: destination callback with gas limit",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "50000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     50000,
			},
			true,
		},
		{
			"success: destination callback with too much gas limit",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "200000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     100000,
			},
			true,
		},
		{
			"failure: invalid packet data",
			func() {
				packetData = []byte("invalid packet data")
			},
			100000,
			types.CallbackData{},
			false,
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		packetUnmarshaler := transfer.IBCModule{}

		callbackData, err := types.GetDestCallbackData(packetUnmarshaler, packetData, tc.remainingGas)

		if tc.expPass {
			suite.Require().NoError(err)
			suite.Require().Equal(tc.expCallbackData, callbackData)
		} else {
			suite.Require().Error(err)
		}
	}
}

func (suite *CallbacksTypesTestSuite) TestGetCallbackDataErrors() {
	// Success cases are tested above. This test case tests extra error case where
	// the packet data can be unmarshaled but the resulting packet data cannot be
	// casted to a CallbackPacketData.

	packetUnmarshaler := MockPacketDataUnmarshaler{}

	// "no unmarshaler error" instructs the MockPacketDataUnmarshaler to return nil nil
	callbackData, err := types.GetCallbackData(packetUnmarshaler, []byte("no unmarshaler error"), 100000, nil, nil)
	suite.Require().Equal(types.CallbackData{}, callbackData)
	suite.Require().ErrorIs(err, types.ErrNotCallbackPacketData)
}
