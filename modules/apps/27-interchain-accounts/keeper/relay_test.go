package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *KeeperTestSuite) TestTrySendTx() {
	var (
		path   *ibctesting.Path
		msg    interface{}
		portID string
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
				portID = path.EndpointA.ChannelConfig.PortID
			}, true,
		},
		{
			"active channel not found", func() {
				amount, _ := sdk.ParseCoinsNormalized("100stake")
				interchainAccountAddr, _ := suite.chainB.GetSimApp().ICAKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID)
				msg = &banktypes.MsgSend{FromAddress: interchainAccountAddr, ToAddress: suite.chainB.SenderAccount.GetAddress().String(), Amount: amount}
				portID = "incorrect portID"
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

			err := InitInterchainAccount(path.EndpointA, owner)
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			suite.Require().NoError(err)

			err = suite.chainB.GetSimApp().ICAKeeper.OnChanOpenConfirm(suite.chainA.GetContext(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
			suite.Require().NoError(err)

			tc.malleate()
			_, err = suite.chainA.GetSimApp().ICAKeeper.TrySendTx(suite.chainA.GetContext(), portID, msg)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
