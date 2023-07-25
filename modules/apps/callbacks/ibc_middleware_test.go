package ibccallbacks_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	ibccallbacks "github.com/cosmos/ibc-go/v7/modules/apps/callbacks"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

func (suite *CallbacksTestSuite) TestNilUnderlyingApp() {
	suite.setupChains()

	channelKeeper := suite.chainA.App.GetIBCKeeper().ChannelKeeper
	mockContractKeeper := suite.chainA.GetSimApp().MockKeeper

	// require panic
	suite.PanicsWithValue(fmt.Sprintf("underlying application does not implement %T", (*types.CallbacksCompatibleModule)(nil)), func() {
		_ = ibccallbacks.NewIBCMiddleware(nil, channelKeeper, mockContractKeeper, uint64(1000000))
	})
}

func (suite *CallbacksTestSuite) TestNilContractKeeper() {
	suite.setupChains()

	channelKeeper := suite.chainA.App.GetIBCKeeper().ChannelKeeper
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	// require panic
	suite.PanicsWithValue("contract keeper cannot be nil", func() {
		_ = ibccallbacks.NewIBCMiddleware(transferStack, channelKeeper, nil, uint64(1000000))
	})
}

func (suite *CallbacksTestSuite) TestNilICS4Wrapper() {
	suite.setupChains()

	mockContractKeeper := suite.chainA.GetSimApp().MockKeeper
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	// require panic
	suite.PanicsWithValue("ics4wrapper cannot be nil", func() {
		_ = ibccallbacks.NewIBCMiddleware(transferStack, nil, mockContractKeeper, uint64(1000000))
	})
}

func (suite *CallbacksTestSuite) TestUnmarshalPacketData() {
	suite.setupChains()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack UnmarshalPacketData call order: callbacks -> fee -> transfer
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	unmarshalerStack, ok := transferStack.(types.CallbacksCompatibleModule)
	suite.Require().True(ok)

	expPacketData := transfertypes.FungibleTokenPacketData{
		Denom:    ibctesting.TestCoin.Denom,
		Amount:   ibctesting.TestCoin.Amount.String(),
		Sender:   ibctesting.TestAccAddress,
		Receiver: ibctesting.TestAccAddress,
		Memo:     fmt.Sprintf(`{"src_callback": {"address": "%s"}, "dest_callback": {"address":"%s"}}`, ibctesting.TestAccAddress, ibctesting.TestAccAddress),
	}
	data := expPacketData.GetBytes()

	packetData, err := unmarshalerStack.UnmarshalPacketData(data)
	suite.Require().NoError(err)
	suite.Require().Equal(expPacketData, packetData)
}

func (suite *CallbacksTestSuite) TestGetAppVersion() {
	suite.SetupICATest()
	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack GetAppVersion call order: callbacks -> fee -> icacontroller
	icaControllerStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	suite.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	appVersion, found := controllerStack.GetAppVersion(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	suite.Require().True(found)
	suite.Require().Equal(suite.path.EndpointA.ChannelConfig.Version, appVersion)
}

func (suite *CallbacksTestSuite) TestOnChanCloseInit() {
	suite.SetupICATest()
	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack OnChanCloseInit call order: callbacks -> fee -> icacontroller
	icaControllerStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	suite.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	err := controllerStack.OnChanCloseInit(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	// we just check that this call is passed down to the icacontroller to return an error
	suite.Require().ErrorIs(errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel"), err)
}

func (suite *CallbacksTestSuite) TestOnChanCloseConfirm() {
	suite.SetupICATest()
	// We will pass the function call down the icacontroller stack to the icacontroller module
	// icacontroller stack OnChanCloseConfirm call order: callbacks -> fee -> icacontroller
	icaControllerStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	suite.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	err := controllerStack.OnChanCloseConfirm(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	// we just check that this call is passed down to the icacontroller
	suite.Require().NoError(err)
}

func (suite *CallbacksTestSuite) TestSendPacket() {
	suite.SetupICATest()
	// We will pass the function call down the icacontroller stack to the channel keeper
	// icacontroller stack SendPacket call order: callbacks -> fee -> channel
	icaControllerStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	suite.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	seq, err := controllerStack.SendPacket(suite.chainA.GetContext(), nil, "invalid_port", "invalid_channel", clienttypes.NewHeight(1, 100), 0, nil)
	// we just check that this call is passed down to the channel keeper to return an error
	suite.Require().Equal(uint64(0), seq)
	suite.Require().ErrorIs(errorsmod.Wrap(channeltypes.ErrChannelNotFound, "invalid_channel"), err)
}

func (suite *CallbacksTestSuite) TestWriteAcknowledgement() {
	suite.SetupTransferTest()

	// build packet
	packetData := transfertypes.NewFungibleTokenPacketData(
		ibctesting.TestCoin.Denom,
		ibctesting.TestCoin.Amount.String(),
		ibctesting.TestAccAddress,
		ibctesting.TestAccAddress,
		fmt.Sprintf(`{"dest_callback": {"address":"%s"}}`, ibctesting.TestAccAddress),
	)

	packet := channeltypes.NewPacket(
		packetData.GetBytes(),
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	transferStack, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	transferStackMw := transferStack.(porttypes.Middleware)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := suite.chainB.GetChannelCapability(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)

	err := transferStackMw.WriteAcknowledgement(suite.chainB.GetContext(), chanCap, packet, ack)
	suite.Require().NoError(err)

	packetAck, _ := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, 1)
	suite.Require().Equal(packetAck, channeltypes.CommitAcknowledgement(ack.Acknowledgement()))
}

func (suite *CallbacksTestSuite) TestWriteAcknowledgementError() {
	suite.SetupICATest()

	packet := channeltypes.NewPacket(
		[]byte("invalid packet data"),
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		"invalid_port",
		"invalid_channel",
		clienttypes.NewHeight(1, 100),
		0,
	)

	icaControllerStack, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	suite.Require().True(ok)

	callbackStack := icaControllerStack.(porttypes.Middleware)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := suite.chainB.GetChannelCapability(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)

	err := callbackStack.WriteAcknowledgement(suite.chainB.GetContext(), chanCap, packet, ack)
	suite.Require().ErrorIs(err, errorsmod.Wrap(channeltypes.ErrChannelNotFound, packet.GetDestChannel()))
}

func (suite *CallbacksTestSuite) TestOnAcknowledgementPacketError() {
	// The successful cases are tested in transfer_test.go and ica_test.go.
	// This test case tests the error case by passing an invalid packet data.
	suite.SetupTransferTest()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack OnAcknowledgementPacket call order: callbacks -> fee -> transfer
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	err := transferStack.OnAcknowledgementPacket(suite.chainA.GetContext(), channeltypes.Packet{}, []byte("invalid"), suite.chainA.SenderAccount.GetAddress())
	suite.Require().ErrorIs(ibcerrors.ErrUnknownRequest, err)
	suite.Require().ErrorContains(err, "cannot unmarshal ICS-20 transfer packet acknowledgement:")
}

func (suite *CallbacksTestSuite) TestOnTimeoutPacketError() {
	// The successful cases are tested in transfer_test.go and ica_test.go.
	// This test case tests the error case by passing an invalid packet data.
	suite.SetupTransferTest()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack OnTimeoutPacket call order: callbacks -> fee -> transfer
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	err := transferStack.OnTimeoutPacket(suite.chainA.GetContext(), channeltypes.Packet{}, suite.chainA.SenderAccount.GetAddress())
	suite.Require().ErrorIs(ibcerrors.ErrUnknownRequest, err)
	suite.Require().ErrorContains(err, "cannot unmarshal ICS-20 transfer packet data:")
}

func (suite *CallbacksTestSuite) TestOnRecvPacketAsyncAck() {
	suite.SetupMockFeeTest()

	module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
	suite.Require().NoError(err)
	cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	suite.Require().True(ok)
	mockFeeCallbackStack, ok := cbs.(porttypes.Middleware)
	suite.Require().True(ok)

	packet := channeltypes.NewPacket(
		ibcmock.MockAsyncPacketData,
		suite.chainA.SenderAccount.GetSequence(),
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)

	ack := mockFeeCallbackStack.OnRecvPacket(suite.chainA.GetContext(), packet, suite.chainA.SenderAccount.GetAddress())
	suite.Require().Nil(ack)
	suite.AssertHasExecutedExpectedCallback("none", true)
}

func (suite *CallbacksTestSuite) TestOnRecvPacketFailedAck() {
	suite.SetupMockFeeTest()

	module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
	suite.Require().NoError(err)
	cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	suite.Require().True(ok)
	mockFeeCallbackStack, ok := cbs.(porttypes.Middleware)
	suite.Require().True(ok)

	packet := channeltypes.NewPacket(
		nil,
		suite.chainA.SenderAccount.GetSequence(),
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(0, 100),
		0,
	)

	ack := mockFeeCallbackStack.OnRecvPacket(suite.chainA.GetContext(), packet, suite.chainA.SenderAccount.GetAddress())
	suite.Require().Equal(ibcmock.MockFailAcknowledgement, ack)
	suite.AssertHasExecutedExpectedCallback("none", true)
}

func (suite *CallbacksTestSuite) TestOnRecvPacketLowRelayerGas() {
	suite.SetupTransferTest()

	// build packet
	packetData := transfertypes.NewFungibleTokenPacketData(
		ibctesting.TestCoin.Denom,
		ibctesting.TestCoin.Amount.String(),
		ibctesting.TestAccAddress,
		ibctesting.TestAccAddress,
		fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"500000"}}`, ibctesting.TestAccAddress),
	)

	packet := channeltypes.NewPacket(
		packetData.GetBytes(),
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	transferStack, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	transferStackMw := transferStack.(porttypes.Middleware)

	modifiedCtx := suite.chainB.GetContext().WithGasMeter(sdk.NewGasMeter(400000))
	suite.Require().PanicsWithValue(sdk.ErrorOutOfGas{Descriptor: "mock write_acknowledgement callback panic"}, func() {
		transferStackMw.OnRecvPacket(modifiedCtx, packet, suite.chainB.SenderAccount.GetAddress())
	})

	// check that it doesn't panic when gas is high enough
	ack := transferStackMw.OnRecvPacket(suite.chainB.GetContext(), packet, suite.chainB.SenderAccount.GetAddress())
	suite.Require().NotNil(ack)
}

func (suite *CallbacksTestSuite) TestWriteAcknowledgementOogError() {
	suite.SetupTransferTest()

	// build packet
	packetData := transfertypes.NewFungibleTokenPacketData(
		ibctesting.TestCoin.Denom,
		ibctesting.TestCoin.Amount.String(),
		ibctesting.TestAccAddress,
		ibctesting.TestAccAddress,
		fmt.Sprintf(`{"dest_callback": {"address":"%s", "gas_limit":"350000"}}`, ibctesting.TestAccAddress),
	)

	packet := channeltypes.NewPacket(
		packetData.GetBytes(),
		1,
		suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID,
		suite.path.EndpointB.ChannelConfig.PortID,
		suite.path.EndpointB.ChannelID,
		clienttypes.NewHeight(1, 100),
		0,
	)

	transferStack, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	transferStackMw := transferStack.(porttypes.Middleware)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := suite.chainB.GetChannelCapability(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)

	modifiedCtx := suite.chainB.GetContext().WithGasMeter(sdk.NewGasMeter(300_000))
	suite.Require().PanicsWithValue(sdk.ErrorOutOfGas{Descriptor: "mock write_acknowledgement callback panic"}, func() {
		_ = transferStackMw.WriteAcknowledgement(modifiedCtx, chanCap, packet, ack)
	})
}

func (suite *CallbacksTestSuite) TestOnAcknowledgementPacketLowRelayerGas() {
	suite.SetupTransferTest()

	senderAddr := suite.chainA.SenderAccount.GetAddress()
	amount := ibctesting.TestCoin
	msg := transfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
		amount, suite.chainA.SenderAccount.GetAddress().String(),
		senderAddr.String(), clienttypes.NewHeight(1, 100), 0,
		fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"350000"}}`, ibctesting.TestAccAddress),
	)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents().ToABCIEvents())
	suite.Require().NoError(err) // packet committed
	suite.Require().NotNil(packet)

	// relay to chainB
	err = suite.path.EndpointB.UpdateClient()
	suite.Require().NoError(err)
	res, err = suite.path.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(res)

	// relay ack to chainA
	ack, err := ibctesting.ParseAckFromEvents(res.Events)
	suite.Require().NoError(err)

	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)
	// Low Relayer gas
	modifiedCtx := suite.chainA.GetContext().WithGasMeter(sdk.NewGasMeter(300_000))
	suite.Require().PanicsWithValue(sdk.ErrorOutOfGas{Descriptor: "mock acknowledgement callback panic"}, func() {
		_ = transferStack.OnAcknowledgementPacket(modifiedCtx, packet, ack, senderAddr)
	})
}

func (suite *CallbacksTestSuite) TestOnTimeoutPacketLowRelayerGas() {
	suite.SetupTransferTest()

	timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
	timeoutTimestamp := uint64(suite.chainB.GetContext().BlockTime().UnixNano())

	amount := ibctesting.TestCoin
	msg := transfertypes.NewMsgTransfer(
		suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID,
		amount, suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, timeoutTimestamp,
		fmt.Sprintf(`{"src_callback": {"address":"%s", "gas_limit":"350000"}}`, ibctesting.TestAccAddress),
	)

	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents().ToABCIEvents())
	suite.Require().NoError(err) // packet committed
	suite.Require().NotNil(packet)

	// need to update chainA's client representing chainB to prove missing ack
	err = suite.path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)
	modifiedCtx := suite.chainA.GetContext().WithGasMeter(sdk.NewGasMeter(300_000))
	suite.Require().PanicsWithValue(sdk.ErrorOutOfGas{Descriptor: "mock timeout callback panic"}, func() {
		_ = transferStack.OnTimeoutPacket(modifiedCtx, packet, suite.chainA.SenderAccount.GetAddress())
	})
}

func (suite *CallbacksTestSuite) TestSendPacketReject() {
	suite.SetupTransferTest()

	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)
	callbackStack, ok := transferStack.(porttypes.Middleware)
	suite.Require().True(ok)

	// We use the MockCallbackUnauthorizedAddress so that mock contract keeper knows to reject the packet
	ftpd := transfertypes.NewFungibleTokenPacketData(
		ibctesting.TestCoin.GetDenom(), ibctesting.TestCoin.Amount.String(), ibcmock.MockCallbackUnauthorizedAddress,
		ibctesting.TestAccAddress, fmt.Sprintf(`{"src_callback": {"address": "%s"}}`, callbackAddr),
	)

	channelCap := suite.path.EndpointA.Chain.GetChannelCapability(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	seq, err := callbackStack.SendPacket(
		suite.chainA.GetContext(), channelCap, suite.path.EndpointA.ChannelConfig.PortID,
		suite.path.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0, ftpd.GetBytes(),
	)
	suite.Require().ErrorIs(err, ibcmock.ErrorMock)
	suite.Require().Equal(uint64(0), seq)
}

func (suite *CallbacksTestSuite) TestProcessCallbackDataGetterError() {
	// The successful cases, other errors, and panics are tested in transfer_test.go and ica_test.go.
	suite.SetupTransferTest()

	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)
	callbackStack, ok := transferStack.(ibccallbacks.IBCMiddleware)
	suite.Require().True(ok)

	invalidDataGetter := func() (types.CallbackData, bool, error) {
		return types.CallbackData{}, false, fmt.Errorf("invalid data getter")
	}
	mockPacket := channeltypes.Packet{Sequence: 0}

	mockLogger := ibcmock.NewMockLogger()
	ctx := suite.chainA.GetContext().WithLogger(mockLogger)

	err := callbackStack.ProcessCallback(ctx, mockPacket, types.CallbackTypeWriteAcknowledgement, invalidDataGetter, nil)
	suite.Require().NoError(err)
	suite.Require().Equal(1, len(mockLogger.DebugLogs))
	suite.Require().Equal("Failed to get callback data.", mockLogger.DebugLogs[0].Message)
	suite.Require().Equal([]interface{}{"packet", mockPacket, "err", fmt.Errorf("invalid data getter")}, mockLogger.DebugLogs[0].Params)
}
