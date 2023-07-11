package ibccallbacks_test

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	ibccallbacks "github.com/cosmos/ibc-go/v7/modules/apps/callbacks"
	"github.com/cosmos/ibc-go/v7/modules/apps/callbacks/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
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
	// transfer stack call order: callbacks -> fee -> transfer
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
	// icacontroller stack call order: callbacks -> fee -> icacontroller
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
	// icacontroller stack call order: callbacks -> fee -> icacontroller
	icaControllerStack, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(icacontrollertypes.SubModuleName)
	suite.Require().True(ok)

	controllerStack := icaControllerStack.(porttypes.Middleware)
	err := controllerStack.OnChanCloseInit(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
	suite.Require().ErrorIs(errorsmod.Wrap(ibcerrors.ErrInvalidRequest, "user cannot close channel"), err)
}
