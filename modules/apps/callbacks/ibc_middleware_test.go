package ibccallbacks_test

import (
	"encoding/json"
	"errors"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	ibccallbacks "github.com/cosmos/ibc-go/v10/modules/apps/callbacks"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/internal"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/testing/simapp"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
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
				_ = ibccallbacks.NewIBCMiddleware(simapp.ContractKeeper{}, maxCallbackGas)
			},
			nil,
		},
		{
			"panics with nil contract keeper",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(nil, maxCallbackGas)
			},
			errors.New("contract keeper cannot be nil"),
		},
		{
			"panics with zero maxCallbackGas",
			func() {
				_ = ibccallbacks.NewIBCMiddleware(simapp.ContractKeeper{}, uint64(0))
			},
			errors.New("maxCallbackGas cannot be zero"),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.expError == nil {
				s.Require().NotPanics(tc.instantiateFn, "unexpected panic: NewIBCMiddleware")
			} else {
				s.Require().PanicsWithError(tc.expError.Error(), tc.instantiateFn, "expected panic with error: ", tc.expError.Error())
			}
		})
	}
}

func (s *CallbacksTestSuite) TestSetICS4Wrapper() {
	s.setupChains()

	cbsMiddleware := ibccallbacks.IBCMiddleware{}
	s.Require().Nil(cbsMiddleware.GetICS4Wrapper())

	s.Require().Panics(func() {
		cbsMiddleware.SetICS4Wrapper(nil)
	}, "expected panic when setting nil ICS4Wrapper")

	cbsMiddleware.SetICS4Wrapper(s.chainA.App.GetIBCKeeper().ChannelKeeper)
	ics4Wrapper := cbsMiddleware.GetICS4Wrapper()

	s.Require().IsType((*channelkeeper.Keeper)(nil), ics4Wrapper)
}

func (s *CallbacksTestSuite) TestSetUnderlyingApplication() {
	s.setupChains()

	cbsMiddleware := ibccallbacks.IBCMiddleware{}

	s.Require().Panics(func() {
		cbsMiddleware.SetUnderlyingApplication(nil)
	}, "expected panic when setting nil underlying application")

	cbsMiddleware.SetUnderlyingApplication(&ibcmock.IBCModule{})

	s.Require().Panics(func() {
		cbsMiddleware.SetUnderlyingApplication(&ibcmock.IBCModule{})
	}, "expected panic when setting underlying application a second time")
}

func (s *CallbacksTestSuite) TestSendPacket() {
	var packetData transfertypes.FungibleTokenPacketData
	var callbackExecuted bool

	testCases := []struct {
		name         string
		malleate     func()
		callbackType types.CallbackType
		expPanic     bool
		expValue     any
	}{
		{
			"success",
			func() {},
			types.CallbackTypeSendPacket,
			false,
			nil,
		},
		{
			"success: callback data doesn't exist",
			func() {
				//nolint:goconst
				packetData.Memo = ""
			},
			"none", // nonexistent callback data should result in no callback execution
			false,
			nil,
		},
		{
			"failure: callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
			},
			"none", // improperly formatted callback data should result in no callback execution
			false,
			types.ErrInvalidCallbackData,
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
		{
			"failure: callback address invalid",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":%d}}`, 50)
				callbackExecuted = false // callback should not be executed
			},
			types.CallbackTypeSendPacket,
			false,
			types.ErrInvalidCallbackData,
		},
		{
			"failure: callback gas limit invalid",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":%d}}`, simapp.SuccessContract, 50)
				callbackExecuted = false // callback should not be executed
			},
			types.CallbackTypeSendPacket,
			false,
			types.ErrInvalidCallbackData,
		},
		{
			"failure: callback calldata invalid",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d", "calldata":%d}}`, simapp.SuccessContract, 50, 50)
				callbackExecuted = false // callback should not be executed
			},
			types.CallbackTypeSendPacket,
			false,
			types.ErrInvalidCallbackData,
		},
		{
			"failure: callback calldata hex invalid",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d", "calldata":"%s"}}`, simapp.SuccessContract, 50, "calldata")
				callbackExecuted = false // callback should not be executed
			},
			types.CallbackTypeSendPacket,
			false,
			types.ErrInvalidCallbackData,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			transferICS4Wrapper := GetSimApp(s.chainA).TransferKeeper.GetICS4Wrapper()

			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				ibctesting.TestAccAddress,
				ibctesting.TestAccAddress,
				fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
			)
			callbackExecuted = true

			tc.malleate()

			ctx := s.chainA.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			var (
				seq uint64
				err error
			)
			sendPacket := func() {
				seq, err = transferICS4Wrapper.SendPacket(ctx, s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, s.chainB.GetTimeoutHeight(), 0, packetData.GetBytes())
			}

			expPass := tc.expValue == nil
			switch {
			case expPass:
				sendPacket()
				s.Require().Nil(err)
				s.Require().Equal(uint64(1), seq)

				expEvent, exists := GetExpectedEvent(
					ctx, transferICS4Wrapper.(porttypes.PacketDataUnmarshaler), gasLimit, packetData.GetBytes(),
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

			if callbackExecuted {
				s.AssertHasExecutedExpectedCallback(tc.callbackType, expPass)
			}
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

	panicError := errors.New("panic error")

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
			"success: callback data doesn't exist",
			func() {
				//nolint:goconst
				packetData.Memo = ""
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			nil,
		},
		{
			"failure: underlying app OnAcknowledgePacket fails",
			func() {
				ack = []byte("invalid ack")
			},
			noExecution,
			ibcerrors.ErrUnknownRequest,
		},
		{
			"failure: no-op on callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			types.ErrInvalidCallbackData,
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
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			userGasLimit = 600000
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				ibctesting.TestAccAddress,
				ibctesting.TestAccAddress,
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
			transferStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
			s.Require().True(ok)

			onAcknowledgementPacket := func() error {
				return transferStack.OnAcknowledgementPacket(ctx, s.path.EndpointA.GetChannel().Version, packet, ack, s.chainA.SenderAccount.GetAddress())
			}

			switch {
			case tc.expError == nil:
				err := onAcknowledgementPacket()
				s.Require().Nil(err)
			case errors.Is(tc.expError, panicError):
				s.Require().PanicsWithValue(storetypes.ErrorOutOfGas{Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeAcknowledgementPacket, userGasLimit)}, func() {
					_ = onAcknowledgementPacket()
				})
			default:
				err := onAcknowledgementPacket()
				s.Require().ErrorIs(err, tc.expError)
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
					ctx, transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data,
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
		expValue  any
	}{
		{
			"success",
			func() {},
			callbackSuccess,
			nil,
		},
		{
			"success: callback data doesn't exist",
			func() {
				//nolint:goconst
				packetData.Memo = ""
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			nil,
		},
		{
			"failure: underlying app OnTimeoutPacket fails",
			func() {
				packet.Data = []byte("invalid packet data")
			},
			noExecution,
			ibcerrors.ErrInvalidType,
		},
		{
			"failure: no-op on callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			types.ErrInvalidCallbackData,
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

			packet, err = ibctesting.ParseV1PacketFromEvents(res.GetEvents())
			s.Require().NoError(err)
			s.Require().NotNil(packet)

			err = json.Unmarshal(packet.Data, &packetData)
			s.Require().NoError(err)

			ctx = s.chainA.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			tc.malleate()

			// callbacks module is routed as top level middleware
			transferStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
			s.Require().True(ok)

			onTimeoutPacket := func() error {
				return transferStack.OnTimeoutPacket(ctx, s.path.EndpointA.GetChannel().Version, packet, s.chainA.SenderAccount.GetAddress())
			}

			switch expValue := tc.expValue.(type) {
			case nil:
				err := onTimeoutPacket()
				s.Require().Nil(err)
			case error:
				err := onTimeoutPacket()
				s.Require().ErrorIs(err, expValue)
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
					ctx, transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data,
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
	panicAck := channeltypes.NewErrorAcknowledgement(errors.New("panic"))

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
			"success: callback data doesn't exist",
			func() {
				//nolint:goconst
				packetData.Memo = ""
				packet.Data = packetData.GetBytes()
			},
			noExecution,
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
			"failure: callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"dest_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			noExecution,
			channeltypes.NewErrorAcknowledgement(types.ErrInvalidCallbackData),
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
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			// set user gas limit above panic level in mock contract keeper
			userGasLimit = 600_000
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				ibctesting.TestAccAddress,
				s.chainB.SenderAccount.GetAddress().String(),
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
			transferStack, ok := s.chainB.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
			s.Require().True(ok)

			onRecvPacket := func() ibcexported.Acknowledgement {
				return transferStack.OnRecvPacket(ctx, s.path.EndpointA.GetChannel().Version, packet, s.chainB.SenderAccount.GetAddress())
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
					ctx, transferStack.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data,
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
			"success: callback data doesn't exist",
			func() {
				//nolint:goconst
				packetData.Memo = ""
				packet.Data = packetData.GetBytes()
			},
			"none", // nonexistent callback data should result in no callback execution
			nil,
		},
		{
			"failure: callback data is not valid",
			func() {
				packetData.Memo = `{"dest_callback": {"address": ""}}`
				packet.Data = packetData.GetBytes()
			},
			"none", // improperly formatted callback data should result in no callback execution
			types.ErrInvalidCallbackData,
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
		s.Run(tc.name, func() {
			s.SetupTransferTest()

			// set user gas limit above panic level in mock contract keeper
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				ibctesting.TestAccAddress,
				s.chainB.SenderAccount.GetAddress().String(),
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

			tc.malleate()

			// callbacks module is routed as top level middleware
			transferICS4Wrapper := GetSimApp(s.chainB).TransferKeeper.GetICS4Wrapper()

			err := transferICS4Wrapper.WriteAcknowledgement(ctx, packet, ack)

			expPass := tc.expError == nil
			s.AssertHasExecutedExpectedCallback(tc.callbackType, expPass)

			if expPass {
				s.Require().NoError(err)

				expEvent, exists := GetExpectedEvent(
					ctx, transferICS4Wrapper.(porttypes.PacketDataUnmarshaler), gasLimit, packet.Data,
					packet.DestinationPort, packet.DestinationChannel, packet.Sequence, types.CallbackTypeReceivePacket, nil,
				)
				if exists {
					s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
				}
			} else {
				s.Require().ErrorIs(err, tc.expError)
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

	callbackError := errors.New("callbackExecutor error")

	testCases := []struct {
		name     string
		malleate func()
		expPanic bool
		expValue any
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
		s.Run(tc.name, func() {
			s.setupChains()

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
			var err error

			processCallback := func() {
				err = internal.ProcessCallback(ctx, callbackType, callbackData, callbackExecutor)
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
				s.Require().ErrorIs(err, tc.expValue.(error))
			}

			s.Require().Equal(expGasConsumed, ctx.GasMeter().GasConsumed())
		})
	}
}

func (s *CallbacksTestSuite) TestUnmarshalPacketDataV1() {
	s.setupChains()
	s.path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	s.path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	s.path.EndpointA.ChannelConfig.Version = transfertypes.V1
	s.path.EndpointB.ChannelConfig.Version = transfertypes.V1
	s.path.Setup()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack UnmarshalPacketData call order: callbacks -> transfer
	transferStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
	s.Require().True(ok)

	unmarshalerStack, ok := transferStack.(porttypes.PacketUnmarshalerModule)
	s.Require().True(ok)

	expPacketDataICS20V1 := transfertypes.FungibleTokenPacketData{
		Denom:    ibctesting.TestCoin.Denom,
		Amount:   ibctesting.TestCoin.Amount.String(),
		Sender:   ibctesting.TestAccAddress,
		Receiver: ibctesting.TestAccAddress,
		Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}, "dest_callback": {"address":"%s"}}`, ibctesting.TestAccAddress, ibctesting.TestAccAddress),
	}

	expPacketDataICS20V2 := transfertypes.InternalTransferRepresentation{
		Token: transfertypes.Token{
			Denom:  transfertypes.NewDenom(ibctesting.TestCoin.Denom),
			Amount: ibctesting.TestCoin.Amount.String(),
		},
		Sender:   ibctesting.TestAccAddress,
		Receiver: ibctesting.TestAccAddress,
		Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}, "dest_callback": {"address":"%s"}}`, ibctesting.TestAccAddress, ibctesting.TestAccAddress),
	}

	portID := s.path.EndpointA.ChannelConfig.PortID
	channelID := s.path.EndpointA.ChannelID

	// Unmarshal ICS20 v1 packet data into v2 packet data
	data := expPacketDataICS20V1.GetBytes()
	packetData, version, err := unmarshalerStack.UnmarshalPacketData(s.chainA.GetContext(), portID, channelID, data)
	s.Require().NoError(err)
	s.Require().Equal(s.path.EndpointA.ChannelConfig.Version, version)
	s.Require().Equal(expPacketDataICS20V2, packetData)
}

func (s *CallbacksTestSuite) TestGetAppVersion() {
	s.SetupICATest()

	// Obtain an IBC stack for testing. The function call will use the top of the stack which calls
	// directly to the channel keeper. Calling from a further down module in the stack is not necessary
	// for this test.
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack, ok := icaControllerStack.(porttypes.ICS4Wrapper)
	s.Require().True(ok)
	appVersion, found := controllerStack.GetAppVersion(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	s.Require().True(found)
	s.Require().Equal(s.path.EndpointA.ChannelConfig.Version, appVersion)
}

func (s *CallbacksTestSuite) TestOnChanCloseInit() {
	s.SetupICATest()

	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack OnChanCloseInit call order: callbacks -> icacontroller
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack, ok := icaControllerStack.(porttypes.Middleware)
	s.Require().True(ok)
	err := controllerStack.OnChanCloseInit(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	// we just check that this call is passed down to the icacontroller to return an error
	s.Require().ErrorIs(err, errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel"))
}

func (s *CallbacksTestSuite) TestOnChanCloseConfirm() {
	s.SetupICATest()

	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack OnChanCloseConfirm call order: callbacks -> icacontroller
	icaControllerStack, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(icacontrollertypes.SubModuleName)
	s.Require().True(ok)

	controllerStack, ok := icaControllerStack.(porttypes.Middleware)
	s.Require().True(ok)
	err := controllerStack.OnChanCloseConfirm(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	// we just check that this call is passed down to the icacontroller
	s.Require().NoError(err)
}

func (s *CallbacksTestSuite) TestOnRecvPacketAsyncAck() {
	s.setupChains()

	cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(ibctesting.MockPort)
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

	ack := cbs.OnRecvPacket(s.chainA.GetContext(), ibcmock.Version, packet, s.chainA.SenderAccount.GetAddress())
	s.Require().Nil(ack)
	s.AssertHasExecutedExpectedCallback("none", true)
}
