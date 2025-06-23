package v2_test

import (
	"errors"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/testing/simapp"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/callbacks/v2"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channelkeeperv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/keeper"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
	ibcmockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
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
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, &channelkeeperv2.Keeper{}, simapp.ContractKeeper{}, &channelkeeperv2.Keeper{}, maxCallbackGas)
			},
			nil,
		},
		{
			"panics with nil ics4wrapper",
			func() {
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, nil, simapp.ContractKeeper{}, &channelkeeperv2.Keeper{}, maxCallbackGas)
			},
			errors.New("write acknowledgement wrapper cannot be nil"),
		},
		{
			"panics with nil underlying app",
			func() {
				_ = v2.NewIBCMiddleware(nil, &channelkeeperv2.Keeper{}, simapp.ContractKeeper{}, &channelkeeperv2.Keeper{}, maxCallbackGas)
			},
			fmt.Errorf("underlying application does not implement %T", (*api.PacketUnmarshalerModuleV2)(nil)),
		},
		{
			"panics with nil contract keeper",
			func() {
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, &channelkeeperv2.Keeper{}, nil, &channelkeeperv2.Keeper{}, maxCallbackGas)
			},
			errors.New("contract keeper cannot be nil"),
		},
		{
			"panics with nil channel v2 keeper",
			func() {
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, &channelkeeperv2.Keeper{}, simapp.ContractKeeper{}, nil, maxCallbackGas)
			},
			errors.New("channel keeper v2 cannot be nil"),
		},
		{
			"panics with zero maxCallbackGas",
			func() {
				_ = v2.NewIBCMiddleware(ibcmockv2.IBCModule{}, &channelkeeperv2.Keeper{}, simapp.ContractKeeper{}, &channelkeeperv2.Keeper{}, uint64(0))
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

func (s *CallbacksTestSuite) TestWithWriteAckWrapper() {
	s.setupChains()

	cbsMiddleware := v2.IBCMiddleware{}
	s.Require().Nil(cbsMiddleware.GetWriteAckWrapper())

	cbsMiddleware.WithWriteAckWrapper(s.chainA.App.GetIBCKeeper().ChannelKeeperV2)
	writeAckWrapper := cbsMiddleware.GetWriteAckWrapper()

	s.Require().IsType((*channelkeeperv2.Keeper)(nil), writeAckWrapper)
}

func (s *CallbacksTestSuite) TestSendPacket() {
	var packetData transfertypes.FungibleTokenPacketData

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
			"success: callback data nonexistent",
			func() {
				//nolint:goconst
				packetData.Memo = ""
			},
			"none",
			false,
			nil,
		},
		{
			"failure: no-op on callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
			},
			"none", // improperly formatted callback data should result in no callback execution
			false,
			types.ErrInvalidCallbackData,
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
		s.Run(tc.name, func() {
			s.SetupTest()

			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				s.chainA.SenderAccount.GetAddress().String(),
				ibctesting.TestAccAddress,
				fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, simapp.SuccessContract),
			)

			tc.malleate()

			payload := channeltypesv2.NewPayload(
				transfertypes.PortID, transfertypes.PortID,
				transfertypes.V1, transfertypes.EncodingJSON,
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
					ctx, packetData, gasLimit, payload.Version,
					transfertypes.PortID, s.path.EndpointA.ClientID, 1, types.CallbackTypeSendPacket, nil,
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

func (s *CallbacksTestSuite) TestOnAcknowledgementPacket() {
	type expResult uint8
	const (
		noExecution expResult = iota
		callbackFailed
		callbackSuccess
	)

	var (
		packetData   transfertypes.FungibleTokenPacketData
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
			"success: callback data nonexistent",
			func() {
				//nolint:goconst
				packetData.Memo = ""
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
			"failure: callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
			},
			noExecution,
			types.ErrInvalidCallbackData,
		},
		{
			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
			},
			callbackFailed,
			nil,
		},
		{
			"failure: callback execution panics on insufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)

				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300_000))
			},
			callbackFailed,
			panicError,
		},
		{
			"failure: callback execution fails",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.ErrorContract)
			},
			callbackFailed,
			nil, // execution failure in OnAcknowledgement should not block acknowledgement processing
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			userGasLimit = 600000
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				ibctesting.TestAccAddress,
				ibctesting.TestAccAddress,
				fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.SuccessContract, userGasLimit),
			)

			ack = channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement()
			ctx = s.chainA.GetContext()

			// may malleate packetData, ack, and ctx
			tc.malleate()

			payload := channeltypesv2.NewPayload(
				transfertypes.PortID, transfertypes.PortID,
				transfertypes.V1, transfertypes.EncodingJSON,
				packetData.GetBytes(),
			)

			gasLimit := ctx.GasMeter().Limit()

			// callbacks module is routed as top level middleware
			cbs := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			onAcknowledgementPacket := func() error {
				return cbs.OnAcknowledgementPacket(ctx, s.path.EndpointA.ClientID, s.path.EndpointB.ClientID, 1, ack, payload, s.chainA.SenderAccount.GetAddress())
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
					ctx, packetData, gasLimit, payload.Version,
					payload.SourcePort, s.path.EndpointA.ClientID, 1, types.CallbackTypeAcknowledgementPacket, nil,
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
			"success: callback data nonexistent",
			func() {
				//nolint:goconst
				packetData.Memo = ""
			},
			noExecution,
			nil,
		},
		{
			"failure: underlying app OnTimeoutPacket fails",
			func() {
				packetData.Amount = "invalid amount"
			},
			noExecution,
			transfertypes.ErrInvalidAmount,
		},
		{
			"failure: callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"src_callback": {"address": ""}}`
			},
			noExecution,
			types.ErrInvalidCallbackData,
		},
		{
			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"400000"}}`, simapp.OogPanicContract)
			},
			callbackFailed,
			nil,
		},
		{
			"failure: callback execution panics on insufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"src_callback": {"address":"%s"}}`, simapp.OogPanicContract)

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
			},
			callbackFailed,
			nil, // execution failure in OnTimeout should not block timeout processing
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			// NOTE: we call send packet so transfer is setup with the correct logic to
			// succeed on timeout
			userGasLimit := 600_000
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Add(time.Second).Unix())
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				s.chainA.SenderAccount.GetAddress().String(),
				ibctesting.TestAccAddress,
				fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.SuccessContract, userGasLimit),
			)

			payload := channeltypesv2.NewPayload(
				transfertypes.PortID, transfertypes.PortID,
				transfertypes.V1, transfertypes.EncodingJSON,
				packetData.GetBytes(),
			)

			packet, err := s.path.EndpointA.MsgSendPacket(timeoutTimestamp, payload)
			s.Require().NoError(err)

			ctx = s.chainA.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			tc.malleate()

			// update packet data in payload after malleate
			payload.Value = packetData.GetBytes()

			// callbacks module is routed as top level middleware
			cbs := s.chainA.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			onTimeoutPacket := func() error {
				return cbs.OnTimeoutPacket(ctx, s.path.EndpointA.ClientID, s.path.EndpointB.ClientID, 1, payload, s.chainA.SenderAccount.GetAddress())
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
					ctx, packetData, gasLimit, payload.Version,
					payload.SourcePort, s.path.EndpointA.ClientID, packet.Sequence, types.CallbackTypeTimeoutPacket, nil,
				)
				s.Require().True(exists)
				s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
			}
		})
	}
}

func (s *CallbacksTestSuite) TestOnRecvPacket() {
	type expResult uint8
	type expRecvStatus uint8
	const (
		noExecution expResult = iota
		callbackFailed
		callbackPanic
		callbackSuccess
	)
	const (
		success expRecvStatus = iota
		panics
		failure
	)

	var (
		packetData   transfertypes.FungibleTokenPacketData
		ctx          sdk.Context
		userGasLimit uint64
	)

	testCases := []struct {
		name          string
		malleate      func()
		expResult     expResult
		expRecvStatus expRecvStatus
	}{
		{
			"success",
			func() {},
			callbackSuccess,
			success,
		},
		{
			"success: callback data nonexistent",
			func() {
				//nolint:goconst
				packetData.Memo = ""
			},
			noExecution,
			success,
		},
		{
			"failure: underlying app OnRecvPacket fails",
			func() {
				packetData.Denom = ""
			},
			noExecution,
			failure,
		},
		{
			"failure: no-op on callback data is not valid",
			func() {
				//nolint:goconst
				packetData.Memo = `{"dest_callback": {"address": ""}}`
			},
			noExecution,
			failure,
		},
		{
			"failure: callback execution reach out of gas, but sufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)
			},
			callbackFailed,
			failure,
		},
		{
			"failure: callback execution panics on insufficient gas provided by relayer",
			func() {
				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, simapp.OogPanicContract, userGasLimit)

				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(300_000))
			},
			callbackFailed,
			panics,
		},
		{
			"failure: callback execution fails",
			func() {
				packetData.Memo = fmt.Sprintf(`{"dest_callback": {"address":"%s"}}`, simapp.ErrorContract)
			},
			callbackFailed,
			failure,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			// set user gas limit above panic level in mock contract keeper
			userGasLimit = 600_000
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				ibctesting.TestAccAddress,
				s.chainB.SenderAccount.GetAddress().String(),
				fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"%d"}}`, ibctesting.TestAccAddress, userGasLimit),
			)

			payload := channeltypesv2.NewPayload(
				transfertypes.PortID, transfertypes.PortID,
				transfertypes.V1, transfertypes.EncodingJSON,
				packetData.GetBytes(),
			)

			ctx = s.chainB.GetContext()
			gasLimit := ctx.GasMeter().Limit()

			tc.malleate()

			// update packet data in payload after malleate
			payload.Value = packetData.GetBytes()

			// callbacks module is routed as top level middleware
			cbs := s.chainB.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)

			onRecvPacket := func() channeltypesv2.RecvPacketResult {
				return cbs.OnRecvPacket(ctx, s.path.EndpointA.ClientID, s.path.EndpointB.ClientID, 1, payload, s.chainB.SenderAccount.GetAddress())
			}

			switch tc.expRecvStatus {
			case success:
				recvResult := onRecvPacket()
				s.Require().Equal(channeltypesv2.PacketStatus_Success, recvResult.Status)

			case panics:
				s.Require().PanicsWithValue(storetypes.ErrorOutOfGas{
					Descriptor: fmt.Sprintf("ibc %s callback out of gas; commitGasLimit: %d", types.CallbackTypeReceivePacket, userGasLimit),
				}, func() {
					_ = onRecvPacket()
				})

			default:
				recvResult := onRecvPacket()
				s.Require().Equal(channeltypesv2.PacketStatus_Failure, recvResult.Status)
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
					ctx, packetData, gasLimit, payload.Version,
					payload.DestinationPort, s.path.EndpointB.ClientID, 1, types.CallbackTypeReceivePacket, nil,
				)
				s.Require().True(exists)
				s.Require().Contains(ctx.EventManager().Events().ToABCIEvents(), expEvent)
			}
		})
	}
}

func (s *CallbacksTestSuite) TestWriteAcknowledgement() {
	var (
		packetData   transfertypes.FungibleTokenPacketData
		destClient   string
		ctx          sdk.Context
		ack          channeltypesv2.Acknowledgement
		multiPayload bool
	)

	successAck := channeltypesv2.NewAcknowledgement(channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement())

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
			"success: callback data nonexistent",
			func() {
				packetData.Memo = ""
				ack = successAck
			},
			"none",
			nil,
		},
		{
			"failure: callback data is not valid",
			func() {
				packetData.Memo = `{"dest_callback": {"address": ""}}`
			},
			"none", // improperly formatted callback data should result in no callback execution
			types.ErrInvalidCallbackData,
		},
		{
			"failure: ics4Wrapper WriteAcknowledgement call fails",
			func() {
				destClient = "invalid-client"
			},
			"none",
			channeltypesv2.ErrInvalidAcknowledgement,
		},
		{
			"failure: multipayload should fail",
			func() {
				multiPayload = true
			},
			"none",
			channeltypesv2.ErrInvalidAcknowledgement,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			// set user gas limit above panic level in mock contract keeper
			packetData = transfertypes.NewFungibleTokenPacketData(
				ibctesting.TestCoin.Denom,
				ibctesting.TestCoin.Amount.String(),
				ibctesting.TestAccAddress,
				s.chainB.SenderAccount.GetAddress().String(),
				fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"600000"}}`, ibctesting.TestAccAddress),
			)

			ctx = s.chainB.GetContext()
			gasLimit := ctx.GasMeter().Limit()
			destClient = s.path.EndpointB.ClientID

			tc.malleate()

			payload := channeltypesv2.NewPayload(
				transfertypes.PortID, transfertypes.PortID,
				transfertypes.V1, transfertypes.EncodingJSON,
				packetData.GetBytes(),
			)
			timeoutTimestamp := uint64(s.chainB.GetContext().BlockTime().Unix())
			var packet channeltypesv2.Packet
			if multiPayload {
				packet = channeltypesv2.NewPacket(
					1, s.path.EndpointA.ClientID, s.path.EndpointB.ClientID,
					timeoutTimestamp, payload, payload,
				)
			} else {
				packet = channeltypesv2.NewPacket(
					1, s.path.EndpointA.ClientID, s.path.EndpointB.ClientID,
					timeoutTimestamp, payload,
				)
			}
			// mock async receive manually so WriteAcknowledgement can pass
			s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetAsyncPacket(ctx, packet.DestinationClient, packet.Sequence, packet)
			s.chainB.App.GetIBCKeeper().ChannelKeeperV2.SetPacketReceipt(ctx, packet.DestinationClient, packet.Sequence)

			// callbacks module is routed as top level middleware
			cbs := s.chainB.App.GetIBCKeeper().ChannelKeeperV2.Router.Route(ibctesting.TransferPort)
			mw, ok := cbs.(api.WriteAcknowledgementWrapper)
			s.Require().True(ok)

			err := mw.WriteAcknowledgement(ctx, destClient, packet.Sequence, ack)

			expPass := tc.expError == nil
			s.AssertHasExecutedExpectedCallback(tc.callbackType, expPass)

			if expPass {
				s.Require().NoError(err)

				expEvent, exists := GetExpectedEvent(
					ctx, packetData, gasLimit, payload.Version,
					payload.DestinationPort, packet.DestinationClient, packet.Sequence, types.CallbackTypeReceivePacket, nil,
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
