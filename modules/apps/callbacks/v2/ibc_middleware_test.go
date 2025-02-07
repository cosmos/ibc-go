package v2_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/ibc-go/modules/apps/callbacks/testing/simapp"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	v2 "github.com/cosmos/ibc-go/modules/apps/callbacks/v2"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	channelkeeperv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/keeper"
	channeltypesv2 "github.com/cosmos/ibc-go/v9/modules/core/04-channel/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	ibcmock "github.com/cosmos/ibc-go/v9/testing/mock"
	ibcmockv2 "github.com/cosmos/ibc-go/v9/testing/mock/v2"
)

func (s *CallbacksTestSuite) TestNewIBCMiddleware() {
	testCases := []struct {
		name          string
		instantiateFn func()
		expError      error
	}{
		{
			"success",
			func() {
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, nil, simapp.ContractKeeper{}, &channelkeeperv2.Keeper{}, maxCallbackGas)
			},
			nil,
		},
		{
			"success with non-nil ics4wrapper",
			func() {
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, &channelkeeperv2.Keeper{}, simapp.ContractKeeper{}, &channelkeeperv2.Keeper{}, maxCallbackGas)
			},
			nil,
		},
		{
			"panics with nil underlying app",
			func() {
				_ = v2.NewIBCMiddleware(nil, nil, simapp.ContractKeeper{}, &channelkeeperv2.Keeper{}, maxCallbackGas)
			},
			fmt.Errorf("underlying application does not implement %T", (*types.CallbacksCompatibleModule)(nil)),
		},
		{
			"panics with nil contract keeper",
			func() {
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, nil, nil, &channelkeeperv2.Keeper{}, maxCallbackGas)
			},
			fmt.Errorf("contract keeper cannot be nil"),
		},
		{
			"panics with zero maxCallbackGas",
			func() {
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, nil, simapp.ContractKeeper{}, &channelkeeperv2.Keeper{}, uint64(0))
			},
			fmt.Errorf("maxCallbackGas cannot be zero"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			if tc.expError == nil {
				s.Require().NotPanics(tc.instantiateFn, "unexpected panic: NewIBCMiddleware")
			} else {
				s.Require().PanicsWithError(tc.expError.Error(), tc.instantiateFn, "expected panic with error: ", tc.expError.Error())
			}
		})
	}
}

func (s *CallbacksTestSuite) TestWithWriteAckWrapper() {
	s.setupChains()

	cbsMiddleware := v2.IBCMiddleware{}
	s.Require().Nil(cbsMiddleware.GetWriteAckWrapper())

	cbsMiddleware.WithWriteAckWrapper(s.chainA.App.GetIBCKeeper().ChannelKeeperV2)
	writeAckWrapper := cbsMiddleware.GetWriteAckWrapper()

	s.Require().IsType((*channelkeeperv2.Keeper)(nil), writeAckWrapper)
}

func (s *CallbacksTestSuite) TestSendPacket() {
	var packetData transfertypes.FungibleTokenPacketDataV2

	testCases := []struct {
		name         string
		malleate     func()
		callbackType types.CallbackType
		expPanic     bool
		expValue     interface{}
	}{
		{
			"success",
			func() {},
			types.CallbackTypeSendPacket,
			false,
			nil,
		},
		{
			"success: multiple denoms",
			func() {
				packetData.Tokens = append(packetData.Tokens, transfertypes.Token{
					Denom:  transfertypes.NewDenom(ibctesting.SecondaryDenom),
					Amount: ibctesting.SecondaryTestCoin.Amount.String(),
				})
			},
			types.CallbackTypeSendPacket,
			false,
			nil,
		},
		{
			"success: no-op on callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
			},
			"none", // improperly formatted callback data should result in no callback execution
			false,
			nil,
		},
		{
			"failure: callback execution fails",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.ErrorContract)
			},
			types.CallbackTypeSendPacket,
			false,
			ibcmock.MockApplicationCallbackError, // execution failure on SendPacket should prevent packet sends
		},
		{
			"failure: callback execution reach out of gas panic, but sufficient gas provided",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"400000"}}`, simapp.OogPanicContract)
			},
			types.CallbackTypeSendPacket,
			true,
			storetypes.ErrorOutOfGas{Descriptor: fmt.Sprintf("mock %s callback oog panic", types.CallbackTypeSendPacket)},
		},
		{
			"failure: callback execution reach out of gas error, but sufficient gas provided",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"400000"}}`, simapp.OogErrorContract)
			},
			types.CallbackTypeSendPacket,
			false,
			errorsmod.Wrapf(types.ErrCallbackOutOfGas, "ibc %s callback out of gas", types.CallbackTypeSendPacket),
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()

			packetData = transfertypes.NewFungibleTokenPacketDataV2(
				[]transfertypes.Token{
					{
						Denom:  transfertypes.NewDenom(ibctesting.TestCoin.Denom),
						Amount: ibctesting.TestCoin.Amount.String(),
					},
				},
				s.chainA.SenderAccount.GetAddress().String(),
				ibctesting.TestAccAddress,
				fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
				ibctesting.EmptyForwardingPacketData,
			)

			tc.malleate()

			payload := channeltypesv2.NewPayload(
				transfertypes.PortID, transfertypes.PortID,
				transfertypes.V2, transfertypes.EncodingProtobuf,
				packetData.GetBytes(),
			)

			ctx := s.chainA.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			var err error
			sendPacket := func() {
				cbs := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

				err = cbs.OnSendPacket(ctx, s.path.EndpointA.ClientID, s.path.EndpointB.ClientID,
					1, payload, s.chainA.SenderAccount.GetAddress())
			}

			expPass := tc.expValue == nil
			switch {
			case expPass:
				sendPacket()
				s.Require().Nil(err)

				expEvent, exists := GetExpectedEvent(
					ctx, gasLimit, packetData.GetBytes(), transfertypes.PortID,
					transfertypes.PortID, s.path.EndpointA.ChannelID, 1, types.CallbackTypeSendPacket, nil,
				)
				if exists {
					s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
				}

			case tc.expPanic:
				s.Require().PanicsWithValue(tc.expValue, sendPacket)

			default:
				sendPacket()
				s.Require().ErrorIs(err, tc.expValue.(error))
			}

			s.AssertHasExecutedExpectedCallback(tc.callbackType, expPass)
		})
	}
}

// func (s *CallbacksTestSuite) TestOnAcknowledgementPacket() {
// 	type expResult uint8
// 	const (
// 		noExecution expResult = iota
// 		callbackFailed
// 		callbackSuccess
// 	)

// 	var (
// 		packetData   transfertypes.FungibleTokenPacketDataV2
// 		packet       channeltypes.Packet
// 		ack          []byte
// 		ctx          sdk.Context
// 		userGasLimit uint64
// 	)

// 	panicError := fmt.Errorf("panic error")

// 	testCases := []struct {
// 		name      string
// 		malleate  func()
// 		expResult expResult
// 		expError  error
// 	}{
// 		{
// 			"success",
// 			func() {},
// 			callbackSuccess,
// 			nil,
// 		},
// 		{
// 			"failure: underlying app OnAcknowledgePacket fails",
// 			func() {
// 				ack = []byte("invalid ack")
// 			},
// 			noExecution,
// 			ibcerrors.ErrUnknownRequest,
// 		},
// 		{
// 			"success: no-op on callback data is not valid",
// 			func() {
// 				//nolint:goconst
// 				packetData.Memo = `{"src_callback": {"address": ""}}`
// 				packet.Data = packetData.GetBytes()
// 			},
// 			noExecution,
// 			nil,
// 		},
// 		{
// 			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
// 				packet.Data = packetData.GetBytes()
// 			},
// 			callbackFailed,
// 			nil,
// 		},
// 		{
// 			"failure: callback execution panics on insufficient gas provided by relayer",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
// 				packet.Data = packetData.GetBytes()

// 				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300_000))
// 			},
// 			callbackFailed,
// 			panicError,
// 		},
// 		{
// 			"failure: callback execution fails",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.ErrorContract)
// 				packet.Data = packetData.GetBytes()
// 			},
// 			callbackFailed,
// 			nil, // execution failure in OnAcknowledgement should not block acknowledgement processing
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc
// 		s.Run(tc.name, func() {
// 			s.SetupTransferTest()

// 			userGasLimit = 600000
// 			packetData = transfertypes.NewFungibleTokenPacketDataV2(
// 				[]transfertypes.Token{
// 					{
// 						Denom:  transfertypes.NewDenom(ibctesting.TestCoin.Denom),
// 						Amount: ibctesting.TestCoin.Amount.String(),
// 					},
// 				},
// 				ibctesting.TestAccAddress,
// 				ibctesting.TestAccAddress,
// 				fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.SuccessContract, userGasLimit),
// 				ibctesting.EmptyForwardingPacketData,
// 			)

// 			packet = channeltypes.Packet{
// 				Sequence:           1,
// 				SourcePort:         s.path.EndpointA.ChannelConfig.PortID,
// 				SourceChannel:      s.path.EndpointA.ChannelID,
// 				DestinationPort:    s.path.EndpointB.ChannelConfig.PortID,
// 				DestinationChannel: s.path.EndpointB.ChannelID,
// 				Data:               packetData.GetBytes(),
// 				TimeoutHeight:      s.chainB.GetTimeoutHeight(),
// 				TimeoutTimestamp:   0,
// 			}

// 			ack = channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement()

// 			ctx = s.chainA.GetContext()
// 			gasLimit := ctx.GasMeter().Limit()

// 			tc.malleate()

// 			// callbacks module is routed as top level middleware
// 			transferStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
// 			s.Require().True(ok)

// 			onAcknowledgementPacket := func() error {
// 				return transferStack.OnAcknowledgementPacket(ctx, s.path.EndpointA.GetChannel().Version, packet, ack, s.chainA.SenderAccount.GetAddress())
// 			}

// 			switch tc.expError {
// 			case nil:
// 				err := onAcknowledgementPacket()
// 				s.Require().Nil(err)

// 			case panicError:
// 				s.Require().PanicsWithValue(storetypes.ErrorOutOfGas{
// 					Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeAcknowledgementPacket, userGasLimit),
// 				}, func() {
// 					_ = onAcknowledgementPacket()
// 				})

// 			default:
// 				err := onAcknowledgementPacket()
// 				s.Require().ErrorIs(err, tc.expError)
// 			}

// 			sourceStatefulCounter := GetSimApp(s.chainA).MockContractKeeper.GetStateEntryCounter(s.chainA.GetContext())
// 			sourceCounters := GetSimApp(s.chainA).MockContractKeeper.Counters

// 			switch tc.expResult {
// 			case noExecution:
// 				s.Require().Len(sourceCounters, 0)
// 				s.Require().Equal(uint8(0), sourceStatefulCounter)

// 			case callbackFailed:
// 				s.Require().Len(sourceCounters, 1)
// 				s.Require().Equal(1, sourceCounters[types.CallbackTypeAcknowledgementPacket])
// 				s.Require().Equal(uint8(0), sourceStatefulCounter)

// 			case callbackSuccess:
// 				s.Require().Len(sourceCounters, 1)
// 				s.Require().Equal(1, sourceCounters[types.CallbackTypeAcknowledgementPacket])
// 				s.Require().Equal(uint8(1), sourceStatefulCounter)

// 				expEvent, exists := GetExpectedEvent(
// 					ctx, transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data, packet.SourcePort,
// 					packet.SourcePort, packet.SourceChannel, packet.Sequence, types.CallbackTypeAcknowledgementPacket, nil,
// 				)
// 				s.Require().True(exists)
// 				s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
// 			}
// 		})
// 	}
// }

// func (s *CallbacksTestSuite) TestOnTimeoutPacket() {
// 	type expResult uint8
// 	const (
// 		noExecution expResult = iota
// 		callbackFailed
// 		callbackSuccess
// 	)

// 	var (
// 		packetData transfertypes.FungibleTokenPacketDataV2
// 		packet     channeltypes.Packet
// 		ctx        sdk.Context
// 	)

// 	testCases := []struct {
// 		name      string
// 		malleate  func()
// 		expResult expResult
// 		expValue  interface{}
// 	}{
// 		{
// 			"success",
// 			func() {},
// 			callbackSuccess,
// 			nil,
// 		},
// 		{
// 			"failure: underlying app OnTimeoutPacket fails",
// 			func() {
// 				packet.Data = []byte("invalid packet data")
// 			},
// 			noExecution,
// 			ibcerrors.ErrInvalidType,
// 		},
// 		{
// 			"success: no-op on callback data is not valid",
// 			func() {
// 				//nolint:goconst
// 				packetData.Memo = `{"src_callback": {"address": ""}}`
// 				packet.Data = packetData.GetBytes()
// 			},
// 			noExecution,
// 			nil,
// 		},
// 		{
// 			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"400000"}}`, simapp.OogPanicContract)
// 				packet.Data = packetData.GetBytes()
// 			},
// 			callbackFailed,
// 			nil,
// 		},
// 		{
// 			"failure: callback execution panics on insufficient gas provided by relayer",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.OogPanicContract)
// 				packet.Data = packetData.GetBytes()

// 				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300_000))
// 			},
// 			callbackFailed,
// 			storetypes.ErrorOutOfGas{
// 				Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeTimeoutPacket, maxCallbackGas),
// 			},
// 		},
// 		{
// 			"failure: callback execution fails",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.ErrorContract)
// 				packet.Data = packetData.GetBytes()
// 			},
// 			callbackFailed,
// 			nil, // execution failure in OnTimeout should not block timeout processing
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc
// 		s.Run(tc.name, func() {
// 			s.SetupTransferTest()

// 			// NOTE: we call send packet so transfer is setup with the correct logic to
// 			// succeed on timeout
// 			userGasLimit := 600_000
// 			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())
// 			msg := transfertypes.NewMsgTransfer(
// 				s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
// 				sdk.NewCoins(ibctesting.TestCoin), s.chainA.SenderAccount.GetAddress().String(),
// 				s.chainB.SenderAccount.GetAddress().String(), clienttypes.ZeroHeight(), timeoutTimestamp,
// 				fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, ibctesting.TestAccAddress, userGasLimit), // set user gas limit above panic level in mock contract keeper
// 				nil,
// 			)

// 			res, err := s.chainA.SendMsgs(msg)
// 			s.Require().NoError(err)
// 			s.Require().NotNil(res)

// 			packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
// 			s.Require().NoError(err)
// 			s.Require().NotNil(packet)

// 			err = proto.Unmarshal(packet.Data, &packetData)
// 			s.Require().NoError(err)

// 			ctx = s.chainA.GetContext()
// 			gasLimit := ctx.GasMeter().Limit()

// 			tc.malleate()

// 			// callbacks module is routed as top level middleware
// 			transferStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
// 			s.Require().True(ok)

// 			onTimeoutPacket := func() error {
// 				return transferStack.OnTimeoutPacket(ctx, s.path.EndpointA.GetChannel().Version, packet, s.chainA.SenderAccount.GetAddress())
// 			}

// 			switch expValue := tc.expValue.(type) {
// 			case nil:
// 				err := onTimeoutPacket()
// 				s.Require().Nil(err)
// 			case error:
// 				err := onTimeoutPacket()
// 				s.Require().ErrorIs(err, expValue)
// 			default:
// 				s.Require().PanicsWithValue(tc.expValue, func() {
// 					_ = onTimeoutPacket()
// 				})
// 			}

// 			sourceStatefulCounter := GetSimApp(s.chainA).MockContractKeeper.GetStateEntryCounter(s.chainA.GetContext())
// 			sourceCounters := GetSimApp(s.chainA).MockContractKeeper.Counters

// 			// account for SendPacket succeeding
// 			switch tc.expResult {
// 			case noExecution:
// 				s.Require().Len(sourceCounters, 1)
// 				s.Require().Equal(uint8(1), sourceStatefulCounter)

// 			case callbackFailed:
// 				s.Require().Len(sourceCounters, 2)
// 				s.Require().Equal(1, sourceCounters[types.CallbackTypeTimeoutPacket])
// 				s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])
// 				s.Require().Equal(uint8(1), sourceStatefulCounter)

// 			case callbackSuccess:
// 				s.Require().Len(sourceCounters, 2)
// 				s.Require().Equal(1, sourceCounters[types.CallbackTypeTimeoutPacket])
// 				s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])
// 				s.Require().Equal(uint8(2), sourceStatefulCounter)

// 				expEvent, exists := GetExpectedEvent(
// 					ctx, transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data, packet.SourcePort,
// 					packet.SourcePort, packet.SourceChannel, packet.Sequence, types.CallbackTypeTimeoutPacket, nil,
// 				)
// 				s.Require().True(exists)
// 				s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
// 			}
// 		})
// 	}
// }

// func (s *CallbacksTestSuite) TestOnRecvPacket() {
// 	type expResult uint8
// 	const (
// 		noExecution expResult = iota
// 		callbackFailed
// 		callbackSuccess
// 	)

// 	var (
// 		packetData   transfertypes.FungibleTokenPacketDataV2
// 		packet       channeltypes.Packet
// 		ctx          sdk.Context
// 		userGasLimit uint64
// 	)

// 	successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
// 	panicAck := channeltypes.NewErrorAcknowledgement(fmt.Errorf("panic"))

// 	testCases := []struct {
// 		name      string
// 		malleate  func()
// 		expResult expResult
// 		expAck    ibcexported.Acknowledgement
// 	}{
// 		{
// 			"success",
// 			func() {},
// 			callbackSuccess,
// 			successAck,
// 		},
// 		{
// 			"failure: underlying app OnRecvPacket fails",
// 			func() {
// 				packet.Data = []byte("invalid packet data")
// 			},
// 			noExecution,
// 			channeltypes.NewErrorAcknowledgement(ibcerrors.ErrInvalidType),
// 		},
// 		{
// 			"success: no-op on callback data is not valid",
// 			func() {
// 				//nolint:goconst
// 				packetData.Memo = `{"dest_callback": {"address": ""}}`
// 				packet.Data = packetData.GetBytes()
// 			},
// 			noExecution,
// 			successAck,
// 		},
// 		{
// 			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
// 				packet.Data = packetData.GetBytes()
// 			},
// 			callbackFailed,
// 			successAck,
// 		},
// 		{
// 			"failure: callback execution panics on insufficient gas provided by relayer",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
// 				packet.Data = packetData.GetBytes()

// 				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300_000))
// 			},
// 			callbackFailed,
// 			panicAck,
// 		},
// 		{
// 			"failure: callback execution fails",
// 			func() {
// 				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s"}}`, simapp.ErrorContract)
// 				packet.Data = packetData.GetBytes()
// 			},
// 			callbackFailed,
// 			successAck,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc
// 		s.Run(tc.name, func() {
// 			s.SetupTransferTest()

// 			// set user gas limit above panic level in mock contract keeper
// 			userGasLimit = 600_000
// 			packetData = transfertypes.NewFungibleTokenPacketDataV2(
// 				[]transfertypes.Token{
// 					{
// 						Denom:  transfertypes.NewDenom(ibctesting.TestCoin.Denom),
// 						Amount: ibctesting.TestCoin.Amount.String(),
// 					},
// 				},
// 				ibctesting.TestAccAddress,
// 				s.chainB.SenderAccount.GetAddress().String(),
// 				fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, ibctesting.TestAccAddress, userGasLimit),
// 				ibctesting.EmptyForwardingPacketData,
// 			)

// 			packet = channeltypes.Packet{
// 				Sequence:           1,
// 				SourcePort:         s.path.EndpointA.ChannelConfig.PortID,
// 				SourceChannel:      s.path.EndpointA.ChannelID,
// 				DestinationPort:    s.path.EndpointB.ChannelConfig.PortID,
// 				DestinationChannel: s.path.EndpointB.ChannelID,
// 				Data:               packetData.GetBytes(),
// 				TimeoutHeight:      s.chainB.GetTimeoutHeight(),
// 				TimeoutTimestamp:   0,
// 			}

// 			ctx = s.chainB.GetContext()
// 			gasLimit := ctx.GasMeter().Limit()

// 			tc.malleate()

// 			// callbacks module is routed as top level middleware
// 			transferStack, ok := s.chainB.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
// 			s.Require().True(ok)

// 			onRecvPacket := func() ibcexported.Acknowledgement {
// 				return transferStack.OnRecvPacket(ctx, s.path.EndpointA.GetChannel().Version, packet, s.chainB.SenderAccount.GetAddress())
// 			}

// 			switch tc.expAck {
// 			case successAck:
// 				ack := onRecvPacket()
// 				s.Require().NotNil(ack)

// 			case panicAck:
// 				s.Require().PanicsWithValue(storetypes.ErrorOutOfGas{
// 					Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeReceivePacket, userGasLimit),
// 				}, func() {
// 					_ = onRecvPacket()
// 				})

// 			default:
// 				ack := onRecvPacket()
// 				s.Require().Equal(tc.expAck, ack)
// 			}

// 			destStatefulCounter := GetSimApp(s.chainB).MockContractKeeper.GetStateEntryCounter(s.chainB.GetContext())
// 			destCounters := GetSimApp(s.chainB).MockContractKeeper.Counters

// 			switch tc.expResult {
// 			case noExecution:
// 				s.Require().Len(destCounters, 0)
// 				s.Require().Equal(uint8(0), destStatefulCounter)

// 			case callbackFailed:
// 				s.Require().Len(destCounters, 1)
// 				s.Require().Equal(1, destCounters[types.CallbackTypeReceivePacket])
// 				s.Require().Equal(uint8(0), destStatefulCounter)

// 			case callbackSuccess:
// 				s.Require().Len(destCounters, 1)
// 				s.Require().Equal(1, destCounters[types.CallbackTypeReceivePacket])
// 				s.Require().Equal(uint8(1), destStatefulCounter)

// 				expEvent, exists := GetExpectedEvent(
// 					ctx, transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data, packet.SourcePort,
// 					packet.DestinationPort, packet.DestinationChannel, packet.Sequence, types.CallbackTypeReceivePacket, nil,
// 				)
// 				s.Require().True(exists)
// 				s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
// 			}
// 		})
// 	}
// }

// func (s *CallbacksTestSuite) TestWriteAcknowledgement() {
// 	var (
// 		packetData transfertypes.FungibleTokenPacketDataV2
// 		packet     channeltypes.Packet
// 		ctx        sdk.Context
// 		ack        ibcexported.Acknowledgement
// 	)

// 	successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

// 	testCases := []struct {
// 		name         string
// 		malleate     func()
// 		callbackType types.CallbackType
// 		expError     error
// 	}{
// 		{
// 			"success",
// 			func() {
// 				ack = successAck
// 			},
// 			types.CallbackTypeReceivePacket,
// 			nil,
// 		},
// 		{
// 			"success: no-op on callback data is not valid",
// 			func() {
// 				packetData.Memo = `{"dest_callback": {"address": ""}}`
// 				packet.Data = packetData.GetBytes()
// 			},
// 			"none", // improperly formatted callback data should result in no callback execution
// 			nil,
// 		},
// 		{
// 			"failure: ics4Wrapper WriteAcknowledgement call fails",
// 			func() {
// 				packet.DestinationChannel = "invalid-channel"
// 			},
// 			"none",
// 			channeltypes.ErrChannelNotFound,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		tc := tc
// 		s.Run(tc.name, func() {
// 			s.SetupTransferTest()

// 			// set user gas limit above panic level in mock contract keeper
// 			packetData = transfertypes.NewFungibleTokenPacketDataV2(
// 				[]transfertypes.Token{
// 					{
// 						Denom:  transfertypes.NewDenom(ibctesting.TestCoin.Denom),
// 						Amount: ibctesting.TestCoin.Amount.String(),
// 					},
// 				},
// 				ibctesting.TestAccAddress,
// 				s.chainB.SenderAccount.GetAddress().String(),
// 				fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"600000"}}`, ibctesting.TestAccAddress),
// 				ibctesting.EmptyForwardingPacketData,
// 			)

// 			packet = channeltypes.Packet{
// 				Sequence:           1,
// 				SourcePort:         s.path.EndpointA.ChannelConfig.PortID,
// 				SourceChannel:      s.path.EndpointA.ChannelID,
// 				DestinationPort:    s.path.EndpointB.ChannelConfig.PortID,
// 				DestinationChannel: s.path.EndpointB.ChannelID,
// 				Data:               packetData.GetBytes(),
// 				TimeoutHeight:      s.chainB.GetTimeoutHeight(),
// 				TimeoutTimestamp:   0,
// 			}

// 			ctx = s.chainB.GetContext()
// 			gasLimit := ctx.GasMeter().Limit()

// 			tc.malleate()

// 			// callbacks module is routed as top level middleware
// 			transferICS4Wrapper := GetSimApp(s.chainB).TransferKeeper.GetICS4Wrapper()

// 			err := transferICS4Wrapper.WriteAcknowledgement(ctx, packet, ack)

// 			expPass := tc.expError == nil
// 			s.AssertHasExecutedExpectedCallback(tc.callbackType, expPass)

// 			if expPass {
// 				s.Require().NoError(err)

// 				expEvent, exists := GetExpectedEvent(
// 					ctx, transferICS4Wrapper.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data, packet.SourcePort,
// 					packet.DestinationPort, packet.DestinationChannel, packet.Sequence, types.CallbackTypeReceivePacket, nil,
// 				)
// 				if exists {
// 					s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
// 				}

// 			} else {
// 				s.Require().ErrorIs(err, tc.expError)
// 			}
// 		})
// 	}
// }
