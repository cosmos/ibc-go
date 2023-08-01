package types_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	transfer "github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (s *CallbacksTypesTestSuite) TestGetSourceCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	var packetData []byte
	// max gas is 1_000_000
	testCases := []struct {
		name            string
		malleate        func()
		remainingGas    uint64
		expCallbackData types.CallbackData
		expAllowRetry   bool
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
				ContractAddr:   sender,
				SenderAddr:     sender,
				GasLimit:       1_000_000,
				CommitGasLimit: 1_000_000,
			},
			false,
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
				ContractAddr:   sender,
				SenderAddr:     sender,
				GasLimit:       50000,
				CommitGasLimit: 50000,
			},
			false,
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
				ContractAddr:   sender,
				SenderAddr:     sender,
				GasLimit:       100000,
				CommitGasLimit: 200000,
			},
			true,
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
				ContractAddr:   sender,
				SenderAddr:     sender,
				GasLimit:       100000,
				CommitGasLimit: 1_000_000,
			},
			true,
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
				ContractAddr:   sender,
				SenderAddr:     sender,
				GasLimit:       1_000_000,
				CommitGasLimit: 1_000_000,
			},
			false,
			true,
		},
		{
			"failure: empty memo",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     "",
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{},
			false,
			false,
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

		testPacket := channeltypes.Packet{Data: packetData}
		callbackData, hasEnoughGas, err := types.GetSourceCallbackData(packetUnmarshaler, testPacket, tc.remainingGas, uint64(1_000_000))

		s.Require().Equal(tc.expAllowRetry, hasEnoughGas, tc.name)
		if tc.expPass {
			s.Require().NoError(err, tc.name)
			s.Require().Equal(tc.expCallbackData, callbackData, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}
	}
}

func (s *CallbacksTypesTestSuite) TestGetDestCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	var packetData []byte
	// max gas is 1_000_000
	testCases := []struct {
		name            string
		malleate        func()
		remainingGas    uint64
		expCallbackData types.CallbackData
		expAllowRetry   bool
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
				ContractAddr:   sender,
				SenderAddr:     "",
				GasLimit:       1_000_000,
				CommitGasLimit: 1_000_000,
			},
			false,
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
				ContractAddr:   sender,
				SenderAddr:     "",
				GasLimit:       50000,
				CommitGasLimit: 50000,
			},
			false,
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
				ContractAddr:   sender,
				SenderAddr:     "",
				GasLimit:       100000,
				CommitGasLimit: 200000,
			},
			true,
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
				ContractAddr:   sender,
				SenderAddr:     "",
				GasLimit:       100000,
				CommitGasLimit: 1_000_000,
			},
			true,
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
				ContractAddr:   sender,
				SenderAddr:     "",
				GasLimit:       1_000_000,
				CommitGasLimit: 1_000_000,
			},
			false,
			true,
		},
		{
			"failure: empty memo",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     "",
				}
				packetData = expPacketData.GetBytes()
			},
			100000,
			types.CallbackData{},
			false,
			false,
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

		testPacket := channeltypes.Packet{Data: packetData}
		callbackData, hasEnoughGas, err := types.GetDestCallbackData(packetUnmarshaler, testPacket, tc.remainingGas, uint64(1_000_000))

		s.Require().Equal(tc.expAllowRetry, hasEnoughGas, tc.name)
		if tc.expPass {
			s.Require().NoError(err, tc.name)
			s.Require().Equal(tc.expCallbackData, callbackData, tc.name)
		} else {
			s.Require().Error(err, tc.name)
		}
	}
}

func (s *CallbacksTypesTestSuite) TestGetCallbackAddress() {
	denom := ibctesting.TestCoin.Denom
	amount := ibctesting.TestCoin.Amount.String()
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	testCases := []struct {
		name       string
		packetData transfertypes.FungibleTokenPacketData
		expAddress string
	}{
		{
			"success: memo has callbacks in json struct and properly formatted src_callback_address which does not match packet sender",
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     "memo",
			},
			"",
		},
		{
			"failure: memo has empty src_callback object",
			transfertypes.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {}}`,
			},
			"",
		},
		{
			"failure: memo does not have callbacks in json struct",
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
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
		s.Run(tc.name, func() {
			callbackData, ok := tc.packetData.GetCustomPacketData(types.SourceCallbackMemoKey).(map[string]interface{})
			s.Require().Equal(ok, callbackData != nil)
			s.Require().Equal(tc.expAddress, types.GetCallbackAddress(callbackData), tc.name)
		})
	}
}

func (s *CallbacksTypesTestSuite) TestUserDefinedGasLimit() {
	denom := ibctesting.TestCoin.Denom
	amount := ibctesting.TestCoin.Amount.String()
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	testCases := []struct {
		name       string
		packetData transfertypes.FungibleTokenPacketData
		expUserGas uint64
	}{
		{
			"success: memo is empty",
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"gas_limit": "100"}}`,
			},
			100,
		},
		{
			"failure: memo has empty src_callback object",
			transfertypes.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {}}`,
			},
			0,
		},
		{
			"failure: memo has user defined gas limit as json number",
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
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
			transfertypes.FungibleTokenPacketData{
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
		callbackData, ok := tc.packetData.GetCustomPacketData(types.SourceCallbackMemoKey).(map[string]interface{})
		s.Require().Equal(ok, callbackData != nil)
		s.Require().Equal(tc.expUserGas, types.GetUserDefinedGasLimit(callbackData), tc.name)
	}
}

func (s *CallbacksTypesTestSuite) TestGetCallbackDataErrors() {
	// Success cases are tested above. This test case tests extra error case where
	// the packet data can be unmarshaled but the resulting packet data cannot be
	// casted to a AdditionalPacketDataProvider.

	packetUnmarshaler := ibcmock.IBCModule{}

	// ibcmock.MockPacketData instructs the MockPacketDataUnmarshaler to return ibcmock.MockPacketData, nil
	mockPacket := channeltypes.Packet{Data: ibcmock.MockPacketData}
	callbackData, allowRetry, err := types.GetCallbackData(packetUnmarshaler, mockPacket, 100000, uint64(1_000_000), types.SourceCallbackMemoKey)
	s.Require().False(allowRetry)
	s.Require().Equal(types.CallbackData{}, callbackData)
	s.Require().ErrorIs(err, types.ErrNotAdditionalPacketDataProvider)
}
