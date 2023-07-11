package ibccallbacks_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	ibccallbacks "github.com/cosmos/ibc-go/v7/modules/apps/callbacks"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
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
		_ = ibccallbacks.NewIBCMiddleware(nil, channelKeeper, mockContractKeeper)
	})
}

func (suite *CallbacksTestSuite) TestUnmarshalPacketData() {
	suite.setupChains()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack UnmarshalPacketData call order: callbacks -> fee -> transfer
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(ibctransfertypes.ModuleName)
	suite.Require().True(ok)

	unmarshalerStack, ok := transferStack.(types.PacketUnmarshalerIBCModule)
	suite.Require().True(ok)

	expPacketData := ibctransfertypes.FungibleTokenPacketData{
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
	icaAddress := suite.SetupICATest()

	// build packet
	packet := suite.buildICAMsgDelegatePacket(icaAddress, clienttypes.NewHeight(1, 100), 0, 1, "")

	icaHostStack, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(icahosttypes.SubModuleName)
	suite.Require().True(ok)

	hostStack := icaHostStack.(porttypes.Middleware)

	ack := channeltypes.NewResultAcknowledgement([]byte("success"))
	chanCap := suite.chainB.GetChannelCapability(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)

	err := hostStack.WriteAcknowledgement(suite.chainB.GetContext(), chanCap, packet, ack)
	suite.Require().NoError(err)

	packetAck, _ := suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.DestinationPort, packet.DestinationChannel, 1)
	suite.Require().Equal(packetAck, channeltypes.CommitAcknowledgement(ack.Acknowledgement()))
}

func (suite *CallbacksTestSuite) TestOnAcknowledgementPacketError() {
	// The successful cases are tested in transfer_test.go and ica_test.go.
	// This test case tests the error case by passing an invalid packet data.
	suite.SetupTransferTest()

	// We will pass the function call down the transfer stack to the transfer module
	// transfer stack OnAcknowledgementPacket call order: callbacks -> fee -> transfer
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(ibctransfertypes.ModuleName)
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
	transferStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(ibctransfertypes.ModuleName)
	suite.Require().True(ok)

	err := transferStack.OnTimeoutPacket(suite.chainA.GetContext(), channeltypes.Packet{}, suite.chainA.SenderAccount.GetAddress())
	suite.Require().ErrorIs(ibcerrors.ErrUnknownRequest, err)
	suite.Require().ErrorContains(err, "cannot unmarshal ICS-20 transfer packet data:")
}
