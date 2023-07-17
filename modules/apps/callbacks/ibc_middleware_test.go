package ibccallbacks_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	ibccallbacks "github.com/cosmos/ibc-go/v7/modules/apps/callbacks"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (suite *CallbacksTestSuite) TestInvalidNewIBCMiddleware() {
	suite.setupChains()

	channelKeeper := suite.chainA.App.GetIBCKeeper().ChannelKeeper
	mockContractKeeper := suite.chainA.GetSimApp().MockKeeper

	// require panic
	suite.Panics(func() {
		_ = ibccallbacks.NewIBCMiddleware(nil, channelKeeper, mockContractKeeper, uint64(1000000))
	})
}

func (suite *CallbacksTestSuite) TestUnmarshalPacketData() {
	suite.setupChains()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack UnmarshalPacketData call order: callbacks -> fee -> transfer
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(transfertypes.ModuleName)
	suite.Require().True(ok)

	unmarshalerStack, ok := transferStack.(types.PacketInfoProviderIBCModule)
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

	icaHostStack, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(icahosttypes.SubModuleName)
	suite.Require().True(ok)

	hostStack := icaHostStack.(porttypes.Middleware)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := suite.chainB.GetChannelCapability(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)

	err := hostStack.WriteAcknowledgement(suite.chainB.GetContext(), chanCap, packet, ack)
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

	ctx := suite.chainA.GetContext()
	mockPacket := channeltypes.Packet{Sequence: 0}
	err := callbackStack.ProcessCallback(ctx, mockPacket, types.CallbackTypeWriteAcknowledgement, invalidDataGetter, nil)
	suite.Require().NoError(err)

	// Verify events
	events := ctx.EventManager().Events().ToABCIEvents()
	suite.T().Log("test: ", events)

	newCtx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	expCallbackData, _, expError := invalidDataGetter()
	types.EmitCallbackEvent(newCtx, mockPacket, types.CallbackTypeWriteAcknowledgement, expCallbackData, expError)
	expEvents := newCtx.EventManager().Events().ToABCIEvents()

	suite.Require().Equal(expEvents, events)
}
