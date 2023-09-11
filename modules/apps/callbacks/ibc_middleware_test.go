package ibccallbacks_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ibccallbacks "github.com/cosmos/ibc-go/modules/apps/callbacks"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/testing/simapp"
	"github.com/cosmos/ibc-go/modules/apps/callbacks/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v8/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v8/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibcmock "github.com/cosmos/ibc-go/v8/testing/mock"
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
				_ = ibccallbacks.NewIBCMiddleware(ibcmock.IBCModule{}, channelkeeper.Keeper{}, simapp.ContractKeeper{}, maxCallbackGas)
			},
			nil,
		},
		{
			"panics with nil underlying app",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(nil, channelkeeper.Keeper{}, simapp.ContractKeeper{}, maxCallbackGas)
			},
			fmt.Errorf("underlying application does not implement %T", (*types.CallbacksCompatibleModule)(nil)),
		},
		{
			"panics with nil contract keeper",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(ibcmock.IBCModule{}, channelkeeper.Keeper{}, nil, maxCallbackGas)
			},
			fmt.Errorf("contract keeper cannot be nil"),
		},
		{
			"panics with nil ics4Wrapper",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(ibcmock.IBCModule{}, nil, simapp.ContractKeeper{}, maxCallbackGas)
			},
			fmt.Errorf("ICS4Wrapper cannot be nil"),
		},
		{
			"panics with zero maxCallbackGas",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(ibcmock.IBCModule{}, channelkeeper.Keeper{}, simapp.ContractKeeper{}, uint64(0))
			},
			fmt.Errorf("maxCallbackGas cannot be zero"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			expPass := tc.expError == nil
			if expPass {
				s.Require().NotPanics(tc.instantiateFn, "unexpected panic: NewIBCMiddleware")
			} else {
				s.Require().PanicsWithError(tc.expError.Error(), tc.instantiateFn, "expected panic with error: ", tc.expError.Error())
			}
		})
	}
}

func (s *CallbacksTestSuite) TestWithICS4Wrapper() {
	s.setupChains()

	cbsMiddleware := ibccallbacks.IBCMiddleware{}
	s.Require().Nil(cbsMiddleware.GetICS4Wrapper())

	cbsMiddleware.WithICS4Wrapper(s.chainA.App.GetIBCKeeper().ChannelKeeper)
	ics4Wrapper := cbsMiddleware.GetICS4Wrapper()

	s.Require().IsType(channelkeeper.Keeper{}, ics4Wrapper)
}

func (s *CallbacksTestSuite) TestSendPacket() {
	var packetData transfertypes.FungibleTokenPacketData

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
			"failure: ics4Wrapper SendPacket call fails",
			func() {
				s.path.EndpointA.ChannelID = "invalid-channel"
			},
			"none", // ics4wrapper failure should result in no callback execution
			false,
			channeltypes.ErrChannelNotFound,
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
			s.SetupTransferTest()

			// callbacks module is routed as top level middleware
			transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
			s.Require().True(ok)

			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.GetDenom(), ibctesting.TestCoin.Amount.String(), ibctesting.TestAccAddress,
				ibctesting.TestAccAddress, fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
			)

			chanCap := s.path.EndpointA.Chain.GetChannelCapability(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)

			tc.malleate()

			ctx := s.chainA.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			var (
				seq uint64
				err error
			)
			sendPacket := func() {
				seq, err = transferStack.(porttypes.ICS4Wrapper).SendPacket(ctx, chanCap, s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, s.chainB.GetTimeoutHeight(), 0, packetData.GetBytes())
			}

			expPass := tc.expValue == nil
			switch {
			case expPass:
				sendPacket()
				s.Require().Nil(err)
				s.Require().Equal(uint64(1), seq)

				expEvent, exists := GetExpectedEvent(
					transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packetData.GetBytes(), s.path.EndpointA.ChannelConfig.PortID,
					s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, seq, types.CallbackTypeSendPacket, nil,
				)
				if exists {
					s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
				}

			case tc.expPanic:
				s.Require().PanicsWithValue(tc.expValue, sendPacket)

			default:
				sendPacket()
				s.Require().ErrorIs(tc.expValue.(error), err)
				s.Require().Equal(uint64(0), seq)
			}

			s.AssertHasExecutedExpectedCallback(tc.callbackType, expPass)
		})
	}
}

func (s *CallbacksTestSuite) TestOnAcknowledgementPacket() {
	type expResult uint8
	const (
		noExecution expResult = iota
		callbackFailed
		callbackSuccess
	)

	var (
		packetData   transfertypes.FungibleTokenPacketData
		packet       channeltypes.Packet
		ack          []byte
		ctx          sdk.Context
		userGasLimit uint64
	)

	panicError := fmt.Errorf("panic error")

	testCases := []struct {
		name      string
		malleate  func()
		expResult expResult
		expError  error
	}{
		{
			"success",
			func() {},
			callbackSuccess,
			nil,
		},
		{
			"failure: underlying app OnAcknolwedgePacket fails",
			func() {
				ack = []byte("invalid ack")
			},
			noExecution,
			ibcerrors.ErrUnknownRequest,
		},
		{
			"success: no-op on callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			nil,
		},
		{
			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
				packet.Data = packetData.GetBytes()
			},
			callbackFailed,
			nil,
		},
		{
			"failure: callback execution panics on insufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
				packet.Data = packetData.GetBytes()

				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300_000))
			},
			callbackFailed,
			panicError,
		},
		{
			"failure: callback execution fails",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.ErrorContract)
				packet.Data = packetData.GetBytes()
			},
			callbackFailed,
			nil, // execution failure in OnAcknowledgement should not block acknowledgement processing
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			userGasLimit = 600000
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.GetDenom(), ibctesting.TestCoin.Amount.String(), ibctesting.TestAccAddress, ibctesting.TestAccAddress,
				fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.SuccessContract, userGasLimit),
			)

			packet = channeltypes.Packet{
				Sequence:           1,
				SourcePort:         s.path.EndpointA.ChannelConfig.PortID,
				SourceChannel:      s.path.EndpointA.ChannelID,
				DestinationPort:    s.path.EndpointB.ChannelConfig.PortID,
				DestinationChannel: s.path.EndpointB.ChannelID,
				Data:               packetData.GetBytes(),
				TimeoutHeight:      s.chainB.GetTimeoutHeight(),
				TimeoutTimestamp:   0,
			}

			ack = channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement()

			ctx = s.chainA.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			tc.malleate()

			// callbacks module is routed as top level middleware
			transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
			s.Require().True(ok)

			onAcknowledgementPacket := func() error {
				return transferStack.OnAcknowledgementPacket(ctx, packet, ack, s.chainA.SenderAccount.GetAddress())
			}

			switch tc.expError {
			case nil:
				err := onAcknowledgementPacket()
				s.Require().Nil(err)

			case panicError:
				s.Require().PanicsWithValue(storetypes.ErrorOutOfGas{
					Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeAcknowledgementPacket, userGasLimit),
				}, func() {
					_ = onAcknowledgementPacket()
				})

			default:
				err := onAcknowledgementPacket()
				s.Require().ErrorIs(tc.expError, err)
			}

			sourceStatefulCounter := GetSimApp(s.chainA).MockContractKeeper.GetStateEntryCounter(s.chainA.GetContext())
			sourceCounters := GetSimApp(s.chainA).MockContractKeeper.Counters

			switch tc.expResult {
			case noExecution:
				s.Require().Len(sourceCounters, 0)
				s.Require().Equal(uint8(0), sourceStatefulCounter)

			case callbackFailed:
				s.Require().Len(sourceCounters, 1)
				s.Require().Equal(1, sourceCounters[types.CallbackTypeAcknowledgementPacket])
				s.Require().Equal(uint8(0), sourceStatefulCounter)

			case callbackSuccess:
				s.Require().Len(sourceCounters, 1)
				s.Require().Equal(1, sourceCounters[types.CallbackTypeAcknowledgementPacket])
				s.Require().Equal(uint8(1), sourceStatefulCounter)

				expEvent, exists := GetExpectedEvent(
					transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data, packet.SourcePort,
					packet.SourcePort, packet.SourceChannel, packet.Sequence, types.CallbackTypeAcknowledgementPacket, nil,
				)
				s.Require().True(exists)
				s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
			}
		})
	}
}

func (s *CallbacksTestSuite) TestOnTimeoutPacket() {
	type expResult uint8
	const (
		noExecution expResult = iota
		callbackFailed
		callbackSuccess
	)

	var (
		packetData transfertypes.FungibleTokenPacketData
		packet     channeltypes.Packet
		ctx        sdk.Context
	)

	testCases := []struct {
		name      string
		malleate  func()
		expResult expResult
		expValue  interface{}
	}{
		{
			"success",
			func() {},
			callbackSuccess,
			nil,
		},
		{
			"failure: underlying app OnTimeoutPacket fails",
			func() {
				packet.Data = []byte("invalid packet data")
			},
			noExecution,
			ibcerrors.ErrUnknownRequest,
		},
		{
			"success: no-op on callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			nil,
		},
		{
			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"400000"}}`, simapp.OogPanicContract)
				packet.Data = packetData.GetBytes()
			},
			callbackFailed,
			nil,
		},
		{
			"failure: callback execution panics on insufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.OogPanicContract)
				packet.Data = packetData.GetBytes()

				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300_000))
			},
			callbackFailed,
			storetypes.ErrorOutOfGas{
				Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeTimeoutPacket, maxCallbackGas),
			},
		},
		{
			"failure: callback execution fails",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.ErrorContract)
				packet.Data = packetData.GetBytes()
			},
			callbackFailed,
			nil, // execution failure in OnTimeout should not block timeout processing
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			// NOTE: we call send packet so transfer is setup with the correct logic to
			// succeed on timeout
			userGasLimit := 600_000
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().UnixNano())
			msg := transfertypes.NewMsgTransfer(
				s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID,
				ibctesting.TestCoin, s.chainA.SenderAccount.GetAddress().String(),
				s.chainB.SenderAccount.GetAddress().String(), clienttypes.ZeroHeight(), timeoutTimestamp,
				fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, ibctesting.TestAccAddress, userGasLimit), // set user gas limit above panic level in mock contract keeper
			)

			res, err := s.chainA.SendMsgs(msg)
			s.Require().NoError(err)
			s.Require().NotNil(res)

			packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
			s.Require().NoError(err)
			s.Require().NotNil(packet)

			err = transfertypes.ModuleCdc.UnmarshalJSON(packet.Data, &packetData)
			s.Require().NoError(err)

			ctx = s.chainA.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			tc.malleate()

			// callbacks module is routed as top level middleware
			transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
			s.Require().True(ok)

			onTimeoutPacket := func() error {
				return transferStack.OnTimeoutPacket(ctx, packet, s.chainA.SenderAccount.GetAddress())
			}

			switch expValue := tc.expValue.(type) {
			case nil:
				err := onTimeoutPacket()
				s.Require().Nil(err)
			case error:
				err := onTimeoutPacket()
				s.Require().ErrorIs(expValue, err)
			default:
				s.Require().PanicsWithValue(tc.expValue, func() {
					_ = onTimeoutPacket()
				})
			}

			sourceStatefulCounter := GetSimApp(s.chainA).MockContractKeeper.GetStateEntryCounter(s.chainA.GetContext())
			sourceCounters := GetSimApp(s.chainA).MockContractKeeper.Counters

			// account for SendPacket succeeding
			switch tc.expResult {
			case noExecution:
				s.Require().Len(sourceCounters, 1)
				s.Require().Equal(uint8(1), sourceStatefulCounter)

			case callbackFailed:
				s.Require().Len(sourceCounters, 2)
				s.Require().Equal(1, sourceCounters[types.CallbackTypeTimeoutPacket])
				s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])
				s.Require().Equal(uint8(1), sourceStatefulCounter)

			case callbackSuccess:
				s.Require().Len(sourceCounters, 2)
				s.Require().Equal(1, sourceCounters[types.CallbackTypeTimeoutPacket])
				s.Require().Equal(1, sourceCounters[types.CallbackTypeSendPacket])
				s.Require().Equal(uint8(2), sourceStatefulCounter)

				expEvent, exists := GetExpectedEvent(
					transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data, packet.SourcePort,
					packet.SourcePort, packet.SourceChannel, packet.Sequence, types.CallbackTypeTimeoutPacket, nil,
				)
				s.Require().True(exists)
				s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
			}
		})
	}
}

func (s *CallbacksTestSuite) TestOnRecvPacket() {
	type expResult uint8
	const (
		noExecution expResult = iota
		callbackFailed
		callbackSuccess
	)

	var (
		packetData   transfertypes.FungibleTokenPacketData
		packet       channeltypes.Packet
		ctx          sdk.Context
		userGasLimit uint64
	)

	successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	panicAck := channeltypes.NewErrorAcknowledgement(fmt.Errorf("panic"))

	testCases := []struct {
		name      string
		malleate  func()
		expResult expResult
		expAck    ibcexported.Acknowledgement
	}{
		{
			"success",
			func() {},
			callbackSuccess,
			successAck,
		},
		{
			"failure: underlying app OnRecvPacket fails",
			func() {
				packet.Data = []byte("invalid packet data")
			},
			noExecution,
			channeltypes.NewErrorAcknowledgement(ibcerrors.ErrInvalidType),
		},
		{
			"success: no-op on callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"dest_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			successAck,
		},
		{
			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
				packet.Data = packetData.GetBytes()
			},
			callbackFailed,
			successAck,
		},
		{
			"failure: callback execution panics on insufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
				packet.Data = packetData.GetBytes()

				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300_000))
			},
			callbackFailed,
			panicAck,
		},
		{
			"failure: callback execution fails",
			func() {
				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s"}}`, simapp.ErrorContract)
				packet.Data = packetData.GetBytes()
			},
			callbackFailed,
			successAck,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			// set user gas limit above panic level in mock contract keeper
			userGasLimit = 600_000
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.GetDenom(), ibctesting.TestCoin.Amount.String(), ibctesting.TestAccAddress, s.chainB.SenderAccount.GetAddress().String(),
				fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, ibctesting.TestAccAddress, userGasLimit),
			)

			packet = channeltypes.Packet{
				Sequence:           1,
				SourcePort:         s.path.EndpointA.ChannelConfig.PortID,
				SourceChannel:      s.path.EndpointA.ChannelID,
				DestinationPort:    s.path.EndpointB.ChannelConfig.PortID,
				DestinationChannel: s.path.EndpointB.ChannelID,
				Data:               packetData.GetBytes(),
				TimeoutHeight:      s.chainB.GetTimeoutHeight(),
				TimeoutTimestamp:   0,
			}

			ctx = s.chainB.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			tc.malleate()

			// callbacks module is routed as top level middleware
			transferStack, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
			s.Require().True(ok)

			onRecvPacket := func() ibcexported.Acknowledgement {
				return transferStack.OnRecvPacket(ctx, packet, s.chainB.SenderAccount.GetAddress())
			}

			switch tc.expAck {
			case successAck:
				ack := onRecvPacket()
				s.Require().NotNil(ack)

			case panicAck:
				s.Require().PanicsWithValue(storetypes.ErrorOutOfGas{
					Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeReceivePacket, userGasLimit),
				}, func() {
					_ = onRecvPacket()
				})

			default:
				ack := onRecvPacket()
				s.Require().Equal(tc.expAck, ack)
			}

			destStatefulCounter := GetSimApp(s.chainB).MockContractKeeper.GetStateEntryCounter(s.chainB.GetContext())
			destCounters := GetSimApp(s.chainB).MockContractKeeper.Counters

			switch tc.expResult {
			case noExecution:
				s.Require().Len(destCounters, 0)
				s.Require().Equal(uint8(0), destStatefulCounter)

			case callbackFailed:
				s.Require().Len(destCounters, 1)
				s.Require().Equal(1, destCounters[types.CallbackTypeReceivePacket])
				s.Require().Equal(uint8(0), destStatefulCounter)

			case callbackSuccess:
				s.Require().Len(destCounters, 1)
				s.Require().Equal(1, destCounters[types.CallbackTypeReceivePacket])
				s.Require().Equal(uint8(1), destStatefulCounter)

				expEvent, exists := GetExpectedEvent(
					transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data, packet.SourcePort,
					packet.DestinationPort, packet.DestinationChannel, packet.Sequence, types.CallbackTypeReceivePacket, nil,
				)
				s.Require().True(exists)
				s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
			}
		})
	}
}

func (s *CallbacksTestSuite) TestWriteAcknowledgement() {
	var (
		packetData transfertypes.FungibleTokenPacketData
		packet     channeltypes.Packet
		ctx        sdk.Context
		ack        ibcexported.Acknowledgement
	)

	successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	testCases := []struct {
		name         string
		malleate     func()
		callbackType types.CallbackType
		expError     error
	}{
		{
			"success",
			func() {
				ack = successAck
			},
			types.CallbackTypeReceivePacket,
			nil,
		},
		{
			"success: no-op on callback data is not valid",
			func() {
				packetData.Memo = `{"dest_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			"none", // improperly formatted callback data should result in no callback execution
			nil,
		},
		{
			"failure: ics4Wrapper WriteAcknowledgement call fails",
			func() {
				packet.DestinationChannel = "invalid-channel"
			},
			"none",
			channeltypes.ErrChannelNotFound,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			// set user gas limit above panic level in mock contract keeper
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.GetDenom(), ibctesting.TestCoin.Amount.String(), ibctesting.TestAccAddress, s.chainB.SenderAccount.GetAddress().String(),
				fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"600000"}}`, ibctesting.TestAccAddress),
			)

			packet = channeltypes.Packet{
				Sequence:           1,
				SourcePort:         s.path.EndpointA.ChannelConfig.PortID,
				SourceChannel:      s.path.EndpointA.ChannelID,
				DestinationPort:    s.path.EndpointB.ChannelConfig.PortID,
				DestinationChannel: s.path.EndpointB.ChannelID,
				Data:               packetData.GetBytes(),
				TimeoutHeight:      s.chainB.GetTimeoutHeight(),
				TimeoutTimestamp:   0,
			}

			ctx = s.chainB.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			chanCap := s.chainB.GetChannelCapability(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)

			tc.malleate()

			// callbacks module is routed as top level middleware
			transferStack, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
			s.Require().True(ok)

			err := transferStack.(porttypes.ICS4Wrapper).WriteAcknowledgement(ctx, chanCap, packet, ack)

			expPass := tc.expError == nil
			s.AssertHasExecutedExpectedCallback(tc.callbackType, expPass)

			if expPass {
				s.Require().NoError(err)

				expEvent, exists := GetExpectedEvent(
					transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data, packet.SourcePort,
					packet.DestinationPort, packet.DestinationChannel, packet.Sequence, types.CallbackTypeReceivePacket, nil,
				)
				if exists {
					s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
				}

			} else {
				s.Require().ErrorIs(tc.expError, err)
			}
		})
	}
}

func (s *CallbacksTestSuite) TestProcessCallback() {
	var (
		callbackType     types.CallbackType
		callbackData     types.CallbackData
		ctx              sdk.Context
		callbackExecutor func(sdk.Context) error
		expGasConsumed   uint64
	)

	callbackError := fmt.Errorf("callbackExecutor error")

	testCases := []struct {
		name     string
		malleate func()
		expPanic bool
		expValue interface{}
	}{
		{
			"success",
			func() {},
			false,
			nil,
		},
		{
			"success: callbackExecutor panic, but not out of gas",
			func() {
				callbackExecutor = func(cachedCtx sdk.Context) error {
					cachedCtx.GasMeter().ConsumeGas(expGasConsumed, "callbackExecutor gas consumption")
					panic("callbackExecutor panic")
				}
			},
			false,
			errorsmod.Wrapf(types.ErrCallbackPanic, "ibc %s callback panicked with: %v", callbackType, "callbackExecutor panic"),
		},
		{
			"success: callbackExecutor oog panic, but retry is not allowed",
			func() {
				executionGas := callbackData.ExecutionGasLimit
				expGasConsumed = executionGas
				callbackExecutor = func(cachedCtx sdk.Context) error { //nolint:unparam
					cachedCtx.GasMeter().ConsumeGas(expGasConsumed+1, "callbackExecutor gas consumption")
					return nil
				}
			},
			false,
			errorsmod.Wrapf(types.ErrCallbackOutOfGas, "ibc %s callback out of gas", callbackType),
		},
		{
			"failure: callbackExecutor error",
			func() {
				callbackExecutor = func(cachedCtx sdk.Context) error {
					cachedCtx.GasMeter().ConsumeGas(expGasConsumed, "callbackExecutor gas consumption")
					return callbackError
				}
			},
			false,
			callbackError,
		},
		{
			"failure: callbackExecutor panic, not out of gas, and SendPacket",
			func() {
				callbackType = types.CallbackTypeSendPacket
				callbackExecutor = func(cachedCtx sdk.Context) error {
					cachedCtx.GasMeter().ConsumeGas(expGasConsumed, "callbackExecutor gas consumption")
					panic("callbackExecutor panic")
				}
			},
			true,
			"callbackExecutor panic",
		},
		{
			"failure: callbackExecutor oog panic, but retry is allowed",
			func() {
				executionGas := callbackData.ExecutionGasLimit
				callbackData.CommitGasLimit = executionGas + 1
				expGasConsumed = executionGas
				callbackExecutor = func(cachedCtx sdk.Context) error { //nolint:unparam
					cachedCtx.GasMeter().ConsumeGas(executionGas+1, "callbackExecutor oog panic")
					return nil
				}
			},
			true,
			storetypes.ErrorOutOfGas{Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeReceivePacket, 1000000+1)},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupMockFeeTest()

			// set a callback data that does not allow retry
			callbackData = types.CallbackData{
				CallbackAddress:   s.chainB.SenderAccount.GetAddress().String(),
				ExecutionGasLimit: 1_000_000,
				SenderAddress:     s.chainB.SenderAccount.GetAddress().String(),
				CommitGasLimit:    600_000,
			}

			// this only makes a difference if it is SendPacket
			callbackType = types.CallbackTypeReceivePacket

			// expGasConsumed can be overwritten in malleate
			expGasConsumed = 300_000

			ctx = s.chainB.GetContext()

			// set a callback executor that will always succeed after consuming expGasConsumed
			callbackExecutor = func(cachedCtx sdk.Context) error { //nolint:unparam
				cachedCtx.GasMeter().ConsumeGas(expGasConsumed, "callbackExecutor gas consumption")
				return nil
			}

			tc.malleate()

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)
			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)
			mockCallbackStack, ok := cbs.(ibccallbacks.IBCMiddleware)
			s.Require().True(ok)

			processCallback := func() {
				err = mockCallbackStack.ProcessCallback(ctx, callbackType, callbackData, callbackExecutor)
			}

			expPass := tc.expValue == nil
			switch {
			case expPass:
				processCallback()
				s.Require().NoError(err)
			case tc.expPanic:
				s.Require().PanicsWithValue(tc.expValue, processCallback)
			default:
				processCallback()
				s.Require().ErrorIs(tc.expValue.(error), err)
			}

			s.Require().Equal(expGasConsumed, ctx.GasMeter().GasConsumed())
		})
	}
}

func (s *CallbacksTestSuite) TestUnmarshalPacketData() {
	s.setupChains()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack UnmarshalPacketData call order: callbacks -> fee -> transfer
	transferStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	s.Require().True(ok)

	unmarshalerStack, ok := transferStack.(types.CallbacksCompatibleModule)
	s.Require().True(ok)

	expPacketData := transfertypes.FungibleTokenPacketData{
		Denom:    ibctesting.TestCoin.Denom,
		Amount:   ibctesting.TestCoin.Amount.String(),
		Sender:   ibctesting.TestAccAddress,
		Receiver: ibctesting.TestAccAddress,
		Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}, "dest_callback": {"address":"%s"}}`, ibctesting.TestAccAddress, ibctesting.TestAccAddress),
	}
	data := expPacketData.GetBytes()

	packetData, err := unmarshalerStack.UnmarshalPacketData(data)
	s.Require().NoError(err)
	s.Require().Equal(expPacketData, packetData)
}

func (s *CallbacksTestSuite) TestGetAppVersion() {
	s.SetupICATest()

	// Obtain an IBC stack for testing. The function call will use the top of the stack which calls
	// directly to the channel keeper. Calling from a further down module in the stack is not necessary
	// for this test.
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.ICS4Wrapper)
	appVersion, found := controllerStack.GetAppVersion(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	s.Require().True(found)
	s.Require().Equal(s.path.EndpointA.ChannelConfig.Version, appVersion)
}

func (s *CallbacksTestSuite) TestOnChanCloseInit() {
	s.SetupICATest()

	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack OnChanCloseInit call order: callbacks -> fee -> icacontroller
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	err := controllerStack.OnChanCloseInit(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	// we just check that this call is passed down to the icacontroller to return an error
	s.Require().ErrorIs(errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel"), err)
}

func (s *CallbacksTestSuite) TestOnChanCloseConfirm() {
	s.SetupICATest()

	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack OnChanCloseConfirm call order: callbacks -> fee -> icacontroller
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	err := controllerStack.OnChanCloseConfirm(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	// we just check that this call is passed down to the icacontroller
	s.Require().NoError(err)
}

func (s *CallbacksTestSuite) TestOnRecvPacketAsyncAck() {
	s.SetupMockFeeTest()

	module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
	s.Require().NoError(err)
	cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	s.Require().True(ok)
	mockFeeCallbackStack, ok := cbs.(porttypes.Middleware)
	s.Require().True(ok)

	packet := channeltypes.NewPacket(
		ibcmock.MockAsyncPacketData,
		s.chainA.SenderAccount.GetSequence(),
		s.path.EndpointA.ChannelConfig.PortID,
		s.path.EndpointA.ChannelID,
		s.path.EndpointB.ChannelConfig.PortID,
		s.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)

	ack := mockFeeCallbackStack.OnRecvPacket(s.chainA.GetContext(), packet, s.chainA.SenderAccount.GetAddress())
	s.Require().Nil(ack)
	s.AssertHasExecutedExpectedCallback("none", true)
}
