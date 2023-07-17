package types_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	transfer "github.com/cosmos/ibc-go/modules/apps/transfer"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *CallbacksTypesTestSuite) TestGetSourceCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	var packetData []byte
	// max gas is 1_000_000
	testCases := []struct {
		name            string
		malleate        func()
		remainingGas    uint64
		expCallbackData types.CallbackData
		expHasEnoughGas bool
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
			2_000_000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     1_000_000,
			},
			true,
			true,
		},
		{
			"success: source callback with gas limit < remaining gas < max gas",
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
			true,
		},
		{
			"success: source callback with remaining gas < gas limit < max gas",
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
			false,
			true,
		},
		{
			"success: source callback with  remaining gas < max gas < gas limit",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "2000000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     100000,
			},
			false,
			true,
		},
		{
			"success: source callback with max gas < remaining gas < gas limit",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "3000000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			2_000_000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     1_000_000,
			},
			true,
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
			false,
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		packetUnmarshaler := transfer.IBCModule{}

		callbackData, hasEnoughGas, err := types.GetSourceCallbackData(packetUnmarshaler, packetData, tc.remainingGas, uint64(1_000_000))

		suite.Require().Equal(tc.expHasEnoughGas, hasEnoughGas, tc.name)
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
			suite.Require().Equal(tc.expCallbackData, callbackData, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}

func (suite *CallbacksTypesTestSuite) TestGetDestCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	var packetData []byte
	// max gas is 1_000_000
	testCases := []struct {
		name            string
		malleate        func()
		remainingGas    uint64
		expCallbackData types.CallbackData
		expHasEnoughGas bool
		expPass         bool
	}{
		{
			"success: dest callback",
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
			2_000_000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     1_000_000,
			},
			true,
			true,
		},
		{
			"success: dest callback with gas limit < remaining gas < max gas",
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
			true,
		},
		{
			"success: dest callback with remaining gas < gas limit < max gas",
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
			false,
			true,
		},
		{
			"success: dest callback with  remaining gas < max gas < gas limit",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "2000000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     100000,
			},
			false,
			true,
		},
		{
			"success: dest callback with max gas < remaining gas < gas limit",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "3000000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			2_000_000,
			types.CallbackData{
				ContractAddr: sender,
				GasLimit:     1_000_000,
			},
			true,
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
			false,
		},
	}

	for _, tc := range testCases {
		tc.malleate()

		packetUnmarshaler := transfer.IBCModule{}

		callbackData, hasEnoughGas, err := types.GetDestCallbackData(packetUnmarshaler, packetData, tc.remainingGas, uint64(1_000_000))

		suite.Require().Equal(tc.expHasEnoughGas, hasEnoughGas, tc.name)
		if tc.expPass {
			suite.Require().NoError(err, tc.name)
			suite.Require().Equal(tc.expCallbackData, callbackData, tc.name)
		} else {
			suite.Require().Error(err, tc.name)
		}
	}
}

func (suite *CallbacksTypesTestSuite) TestGetCallbackDataErrors() {
	// Success cases are tested above. This test case tests extra error case where
	// the packet data can be unmarshaled but the resulting packet data cannot be
	// casted to a CallbackPacketData.

	packetUnmarshaler := MockPacketDataUnmarshaler{}

	// "no unmarshaler error" instructs the MockPacketDataUnmarshaler to return nil nil
	callbackData, hasEnoughGas, err := types.GetCallbackData(packetUnmarshaler, []byte("no unmarshaler error"), 100000, uint64(1_000_000), nil, nil)
	suite.Require().False(hasEnoughGas)
	suite.Require().Equal(types.CallbackData{}, callbackData)
	suite.Require().ErrorIs(err, types.ErrNotCallbackPacketData)
}
