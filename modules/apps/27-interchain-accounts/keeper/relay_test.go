package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *KeeperTestSuite) TestTrySendTx() {
	var (
		path   *ibctesting.Path
		msg    interface{}
		portID string
		appCap *capabilitytypes.Capability
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
			}, true,
		},
		{
			"success with []sdk.Message", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg1 := &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				msg2 := &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				msg = []sdk.Msg{msg1, msg2}
			}, true,
		},
		{
			"incorrect outgoing data", func() {
				msg = []byte{}
			}, false,
		},
		{
			"active channel not found", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				portID = "incorrect portID"
			}, false,
		},
		{
			"channel does not exist", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				suite.chainA.GetSimApp().ICAKeeper.SetActiveChannel(suite.chainA.GetContext(), portID, "channel-100")
			}, false,
		},
		{
			"could not authenticate app capability", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				appCap = nil
			}, false,
		},

		{
			"data is nil", func() {
				msg = nil
			}, false,
		},
		{
			"data is not an SDK message", func() {
				msg = "not an sdk message"
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			owner := "owner"
			suite.coordinator.SetupConnections(path)

			err := suite.SetupICAPath(path, owner)
			suite.Require().NoError(err)
			portID = path.EndpointA.ChannelConfig.PortID
			appCap, _ = suite.chainA.GetSimApp().ScopedICAKeeper.GetCapability(suite.chainA.GetContext(), types.AppCapabilityName(portID, path.EndpointA.ChannelID))
			tc.malleate()
			_, err = suite.chainA.GetSimApp().ICAKeeper.TrySendTx(suite.chainA.GetContext(), appCap, portID, msg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
