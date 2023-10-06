package types_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	transfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
)

func (s *CallbacksTypesTestSuite) TestGetCallbackData() {
	var (
		sender                = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
		receiver              = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
		packetDataUnmarshaler porttypes.PacketDataUnmarshaler
		packetData            []byte
		remainingGas          uint64
		callbackKey           string
	)

	// max gas is 1_000_000
	testCases := []struct {
		name            string
		malleate        func()
		expCallbackData types.CallbackData
		expError        error
	}{
		{
			"success: source callback",
			func() {
				remainingGas = 2_000_000
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{
				CallbackAddress:   sender,
				SenderAddress:     sender,
				ExecutionGasLimit: 1_000_000,
				CommitGasLimit:    1_000_000,
			},
			nil,
		},
		{
			"success: destination callback",
			func() {
				callbackKey = types.DestinationCallbackKey
				remainingGas = 2_000_000
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{
				CallbackAddress:   sender,
				SenderAddress:     "",
				ExecutionGasLimit: 1_000_000,
				CommitGasLimit:    1_000_000,
			},
			nil,
		},
		{
			"success: destination callback with 0 user defined gas limit",
			func() {
				callbackKey = types.DestinationCallbackKey
				remainingGas = 2_000_000
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit":"0"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{
				CallbackAddress:   sender,
				SenderAddress:     "",
				ExecutionGasLimit: 1_000_000,
				CommitGasLimit:    1_000_000,
			},
			nil,
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

				remainingGas = 100_000
			},
			types.CallbackData{
				CallbackAddress:   sender,
				SenderAddress:     sender,
				ExecutionGasLimit: 50_000,
				CommitGasLimit:    50_000,
			},
			nil,
		},
		{
			"success: source callback with remaining gas < gas limit < max gas",
			func() {
				remainingGas = 100_000
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{
				CallbackAddress:   sender,
				SenderAddress:     sender,
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    200_000,
			},
			nil,
		},
		{
			"success: source callback with remaining gas < max gas < gas limit",
			func() {
				remainingGas = 100_000
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "2000000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{
				CallbackAddress:   sender,
				SenderAddress:     sender,
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    1_000_000,
			},
			nil,
		},
		{
			"success: destination callback with remaining gas < max gas < gas limit",
			func() {
				callbackKey = types.DestinationCallbackKey
				remainingGas = 100_000
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "2000000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{
				CallbackAddress:   sender,
				SenderAddress:     "",
				ExecutionGasLimit: 100_000,
				CommitGasLimit:    1_000_000,
			},
			nil,
		},
		{
			"success: source callback with max gas < remaining gas < gas limit",
			func() {
				remainingGas = 2_000_000
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "3000000"}}`, sender),
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{
				CallbackAddress:   sender,
				SenderAddress:     sender,
				ExecutionGasLimit: 1_000_000,
				CommitGasLimit:    1_000_000,
			},
			nil,
		},
		{
			"failure: invalid packet data",
			func() {
				packetData = []byte("invalid packet data")
			},
			types.CallbackData{},
			types.ErrCannotUnmarshalPacketData,
		},
		{
			"failure: packet data does not implement PacketDataProvider",
			func() {
				packetData = ibcmock.MockPacketData
				packetDataUnmarshaler = ibcmock.IBCModule{}
			},
			types.CallbackData{},
			types.ErrNotPacketDataProvider,
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
			types.CallbackData{},
			types.ErrCallbackKeyNotFound,
		},
		{
			"failure: empty address",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     `{"src_callback": {"address": ""}}`,
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{},
			types.ErrCallbackAddressNotFound,
		},
		{
			"failure: space address",
			func() {
				expPacketData := transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     `{"src_callback": {"address": " "}}`,
				}
				packetData = expPacketData.GetBytes()
			},
			types.CallbackData{},
			types.ErrCallbackAddressNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			callbackKey = types.SourceCallbackKey

			packetDataUnmarshaler = transfer.IBCModule{}

			tc.malleate()

			callbackData, err := types.GetCallbackData(packetDataUnmarshaler, packetData, ibcmock.PortID, remainingGas, uint64(1_000_000), callbackKey)

			expPass := tc.expError == nil
			if expPass {
				s.Require().NoError(err, tc.name)
				s.Require().Equal(tc.expCallbackData, callbackData, tc.name)

				expAllowRetry := tc.expCallbackData.ExecutionGasLimit < tc.expCallbackData.CommitGasLimit
				s.Require().Equal(expAllowRetry, callbackData.AllowRetry(), tc.name)
			} else {
				s.Require().ErrorIs(err, tc.expError, tc.name)
			}
		})
	}
}

func (s *CallbacksTypesTestSuite) TestGetSourceCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	packetData := transfertypes.FungibleTokenPacketData{
		Denom:    ibctesting.TestCoin.Denom,
		Amount:   ibctesting.TestCoin.Amount.String(),
		Sender:   sender,
		Receiver: receiver,
		Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, sender),
	}
	packetDataBytes := packetData.GetBytes()

	expCallbackData := types.CallbackData{
		CallbackAddress:   sender,
		SenderAddress:     sender,
		ExecutionGasLimit: 1_000_000,
		CommitGasLimit:    1_000_000,
	}

	packetUnmarshaler := transfer.IBCModule{}

	callbackData, err := types.GetSourceCallbackData(packetUnmarshaler, packetDataBytes, ibcmock.PortID, 2_000_000, 1_000_000)
	s.Require().NoError(err)
	s.Require().Equal(expCallbackData, callbackData)
}

func (s *CallbacksTypesTestSuite) TestGetDestCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	packetData := transfertypes.FungibleTokenPacketData{
		Denom:    ibctesting.TestCoin.Denom,
		Amount:   ibctesting.TestCoin.Amount.String(),
		Sender:   sender,
		Receiver: receiver,
		Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, sender),
	}
	packetDataBytes := packetData.GetBytes()

	expCallbackData := types.CallbackData{
		CallbackAddress:   sender,
		SenderAddress:     "",
		ExecutionGasLimit: 1_000_000,
		CommitGasLimit:    1_000_000,
	}

	packetUnmarshaler := transfer.IBCModule{}

	callbackData, err := types.GetDestCallbackData(packetUnmarshaler, packetDataBytes, ibcmock.PortID, 2_000_000, 1_000_000)
	s.Require().NoError(err)
	s.Require().Equal(expCallbackData, callbackData)
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
			callbackData, ok := tc.packetData.GetCustomPacketData(types.SourceCallbackKey).(map[string]interface{})
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
			"success: user defined gas limit is zero",
			transfertypes.FungibleTokenPacketData{
				Denom:    denom,
				Amount:   amount,
				Sender:   sender,
				Receiver: receiver,
				Memo:     `{"src_callback": {"gas_limit": "0"}}`,
			},
			0,
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
		tc := tc
		s.Run(tc.name, func() {
			callbackData, ok := tc.packetData.GetCustomPacketData(types.SourceCallbackKey).(map[string]interface{})
			s.Require().Equal(ok, callbackData != nil)
			s.Require().Equal(tc.expUserGas, types.GetUserDefinedGasLimit(callbackData), tc.name)
		})
	}
}
