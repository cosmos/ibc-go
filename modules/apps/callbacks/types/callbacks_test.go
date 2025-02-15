package types_test

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

func (s *CallbacksTypesTestSuite) TestGetCallbackData() {
	var (
		sender       = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
		receiver     = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
		packetData   interface{}
		remainingGas uint64
		callbackKey  string
		version      string
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
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      sender,
				ExecutionGasLimit:  1_000_000,
				CommitGasLimit:     1_000_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"success: destination callback",
			func() {
				callbackKey = types.DestinationCallbackKey
				version = transfertypes.V1

				remainingGas = 2_000_000
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      "",
				ExecutionGasLimit:  1_000_000,
				CommitGasLimit:     1_000_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"success: destination callback with 0 user defined gas limit",
			func() {
				callbackKey = types.DestinationCallbackKey
				version = transfertypes.V1

				remainingGas = 2_000_000
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit":"0"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      "",
				ExecutionGasLimit:  1_000_000,
				CommitGasLimit:     1_000_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"success: source callback with gas limit < remaining gas < max gas",
			func() {
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "50000"}}`, sender),
				}

				remainingGas = 100_000
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      sender,
				ExecutionGasLimit:  50_000,
				CommitGasLimit:     50_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"success: source callback with remaining gas < gas limit < max gas",
			func() {
				remainingGas = 100_000
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "200000"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      sender,
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     200_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"success: source callback with remaining gas < max gas < gas limit",
			func() {
				remainingGas = 100_000
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "2000000"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      sender,
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     1_000_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"success: destination callback with remaining gas < max gas < gas limit",
			func() {
				callbackKey = types.DestinationCallbackKey
				version = transfertypes.V1

				remainingGas = 100_000
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s", "gas_limit": "2000000"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      "",
				ExecutionGasLimit:  100_000,
				CommitGasLimit:     1_000_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"success: source callback with max gas < remaining gas < gas limit",
			func() {
				remainingGas = 2_000_000
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s", "gas_limit": "3000000"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      sender,
				ExecutionGasLimit:  1_000_000,
				CommitGasLimit:     1_000_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"failure: packet data does not implement PacketDataProvider",
			func() {
				packetData = ibcmock.MockPacketData
			},
			types.CallbackData{},
			types.ErrNotPacketDataProvider,
		},
		{
			"failure: empty memo",
			func() {
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     "",
				}
			},
			types.CallbackData{},
			types.ErrCallbackKeyNotFound,
		},
		{
			"failure: empty address",
			func() {
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     `{"src_callback": {"address": ""}}`,
				}
			},
			types.CallbackData{},
			types.ErrCallbackAddressNotFound,
		},
		{
			"failure: space address",
			func() {
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     `{"src_callback": {"address": " "}}`,
				}
			},
			types.CallbackData{},
			types.ErrCallbackAddressNotFound,
		},

		{
			"success: source callback",
			func() {
				remainingGas = 2_000_000
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      sender,
				ExecutionGasLimit:  1_000_000,
				CommitGasLimit:     1_000_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"success: destination callback",
			func() {
				callbackKey = types.DestinationCallbackKey

				remainingGas = 2_000_000
				version = transfertypes.V1
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, sender),
				}
			},
			types.CallbackData{
				CallbackAddress:    sender,
				SenderAddress:      "",
				ExecutionGasLimit:  1_000_000,
				CommitGasLimit:     1_000_000,
				ApplicationVersion: transfertypes.V1,
			},
			nil,
		},
		{
			"failure: packet data does not implement PacketDataProvider",
			func() {
				packetData = ibcmock.MockPacketData
			},
			types.CallbackData{},
			types.ErrNotPacketDataProvider,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()

			callbackKey = types.SourceCallbackKey
			version = transfertypes.V1

			tc.malleate()

			callbackData, err := types.GetCallbackData(packetData, version, transfertypes.PortID, remainingGas, uint64(1_000_000), callbackKey)

			if tc.expError == nil {
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

// bytesProvider defines an interface that both packet data types implement in order to fetch the bytes.
type bytesProvider interface {
	GetBytes() []byte
}

func (s *CallbacksTypesTestSuite) TestGetDestSourceCallbackDataTransfer() {
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	var (
		packetData      bytesProvider
		expCallbackData types.CallbackData
	)

	expSrcCallBack := types.CallbackData{
		CallbackAddress:    sender,
		SenderAddress:      sender,
		ExecutionGasLimit:  1_000_000,
		CommitGasLimit:     1_000_000,
		ApplicationVersion: transfertypes.V1,
	}

	expDstCallBack := types.CallbackData{
		CallbackAddress:    sender,
		SenderAddress:      "",
		ExecutionGasLimit:  1_000_000,
		CommitGasLimit:     1_000_000,
		ApplicationVersion: transfertypes.V1,
	}

	testCases := []struct {
		name       string
		malleate   func()
		callbackFn func(
			ctx sdk.Context,
			packetDataUnmarshaler porttypes.PacketDataUnmarshaler,
			packet channeltypes.Packet,
			maxGas uint64,
		) (types.CallbackData, error)
		getSrc bool
	}{
		{
			"success: src_callback v1",
			func() {
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, sender),
				}

				expCallbackData = expSrcCallBack

				s.path.EndpointA.ChannelConfig.Version = transfertypes.V1
				s.path.EndpointA.ChannelConfig.PortID = transfertypes.ModuleName
				s.path.EndpointB.ChannelConfig.Version = transfertypes.V1
				s.path.EndpointB.ChannelConfig.PortID = transfertypes.ModuleName
			},
			types.GetSourceCallbackData,
			true,
		},
		{
			"success: dest_callback v1",
			func() {
				packetData = transfertypes.FungibleTokenPacketData{
					Denom:    ibctesting.TestCoin.Denom,
					Amount:   ibctesting.TestCoin.Amount.String(),
					Sender:   sender,
					Receiver: receiver,
					Memo:     fmt.Sprintf(`{"dest_callback": {"address": "%s"}}`, sender),
				}

				expCallbackData = expDstCallBack

				s.path.EndpointA.ChannelConfig.Version = transfertypes.V1
				s.path.EndpointA.ChannelConfig.PortID = transfertypes.ModuleName
				s.path.EndpointB.ChannelConfig.Version = transfertypes.V1
				s.path.EndpointB.ChannelConfig.PortID = transfertypes.ModuleName
			},
			types.GetDestCallbackData,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()

			tc.malleate()

			transferStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
			s.Require().True(ok)

			packetUnmarshaler, ok := transferStack.(types.CallbacksCompatibleModule)
			s.Require().True(ok)

			s.path.Setup()

			gasMeter := storetypes.NewGasMeter(2_000_000)
			ctx := s.chainA.GetContext().WithGasMeter(gasMeter)
			var packet channeltypes.Packet
			if tc.getSrc {
				packet = channeltypes.NewPacket(packetData.GetBytes(), 0, transfertypes.PortID, s.path.EndpointA.ChannelID, transfertypes.PortID, s.path.EndpointB.ChannelID, clienttypes.ZeroHeight(), 0)
			} else {
				packet = channeltypes.NewPacket(packetData.GetBytes(), 0, transfertypes.PortID, s.path.EndpointB.ChannelID, transfertypes.PortID, s.path.EndpointA.ChannelID, clienttypes.ZeroHeight(), 0)
			}
			callbackData, err := tc.callbackFn(ctx, packetUnmarshaler, packet, 1_000_000)
			s.Require().NoError(err)
			s.Require().Equal(expCallbackData, callbackData)
		})
	}
}

func (s *CallbacksTypesTestSuite) TestGetCallbackAddress() {
	denom := transfertypes.NewDenom(ibctesting.TestCoin.Denom)
	amount := ibctesting.TestCoin.Amount.String()
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	testCases := []struct {
		name       string
		packetData ibcexported.PacketDataProvider
		expAddress string
	}{
		{
			"success: memo has callbacks in json struct and properly formatted src_callback_address which does not match packet sender",
			transfertypes.FungibleTokenPacketData{
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
	denom := transfertypes.NewDenom(ibctesting.TestCoin.Denom)
	amount := ibctesting.TestCoin.Amount.String()
	sender := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	receiver := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	testCases := []struct {
		name       string
		packetData ibcexported.PacketDataProvider
		expUserGas uint64
	}{
		{
			"success: memo is empty",
			transfertypes.FungibleTokenPacketData{
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
				Denom:    denom.Base,
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
