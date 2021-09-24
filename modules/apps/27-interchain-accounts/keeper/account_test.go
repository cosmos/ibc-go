package keeper_test

import (
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v2/testing"
)

func (suite *KeeperTestSuite) TestInitInterchainAccount() {
	var (
		owner string
		path  *ibctesting.Path
		err   error
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{

		{
			"success", func() {}, true,
		},
		/*
		   // TODO: https://github.com/cosmos/ibc-go/issues/288
		   {
		   			"port is already bound", func() {
		   				// mock init interchain account
		   				portID := suite.chainA.GetSimApp().ICAKeeper.GeneratePortId(owner, path.EndpointA.ConnectionID)
		   				suite.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(suite.chainA.GetContext(), portID)
		   			}, false,
		   		},
		*/
		{
			"MsgChanOpenInit fails - channel is already active", func() {
				portID, err := types.GeneratePortID(owner, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)
				suite.Require().NoError(err)
				suite.chainA.GetSimApp().ICAKeeper.SetActiveChannel(suite.chainA.GetContext(), portID, path.EndpointA.ChannelID)
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()        // reset
			owner = TestOwnerAddress // must be explicitly changed
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			tc.malleate() // explicitly change fields in channel and testChannel

			err = suite.chainA.GetSimApp().ICAKeeper.InitInterchainAccount(suite.chainA.GetContext(), path.EndpointA.ConnectionID, path.EndpointB.ConnectionID, owner)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}
