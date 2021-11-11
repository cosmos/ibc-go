package keeper_test

import (
	"github.com/cosmos/ibc-go/v2/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v2/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v2/modules/core/24-host"
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
		{
			"port is already bound",
			func() {
				suite.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(suite.chainA.GetContext(), TestPortID)
			},
			false,
		},
		{
			"fails to generate port-id",
			func() {
				owner = ""
			},
			false,
		},
		{
			"MsgChanOpenInit fails - channel is already active",
			func() {
				portID, err := types.GeneratePortID(owner, path.EndpointA.ConnectionID, path.EndpointB.ConnectionID)
				suite.Require().NoError(err)
				suite.chainA.GetSimApp().ICAKeeper.SetActiveChannelID(suite.chainA.GetContext(), portID, path.EndpointA.ChannelID)
			},
			false,
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

func (suite *KeeperTestSuite) TestInitChannel() {
	var (
		path   *ibctesting.Path
		portID string
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"port is not bound", func() {
				portID = "invalid-portID"
			}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)
			portID, _ = types.GeneratePortID(TestOwnerAddress, path.EndpointA.ConnectionID, path.EndpointA.Counterparty.ConnectionID)

			// bind port and claim capability
			cap := suite.chainA.GetSimApp().ICAKeeper.BindPort(suite.chainA.GetContext(), portID)
			err := suite.chainA.GetSimApp().ICAKeeper.ClaimCapability(suite.chainA.GetContext(), cap, host.PortPath(portID))
			suite.Require().NoError(err)

			tc.malleate() // explicitly change fields in channel and testChannel

			// get next channel seq
			channelSequence := path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(path.EndpointA.Chain.GetContext())

			// Init new channel
			err = suite.chainA.GetSimApp().ICAKeeper.InitChannel(suite.chainA.GetContext(), portID, path.EndpointA.ConnectionID)

			if tc.expPass {
				suite.Require().NoError(err)

				// finish the channel handshake

				// commit state changes for proof verification
				path.EndpointA.Chain.App.Commit()
				path.EndpointA.Chain.NextBlock()

				// update port/channel ids
				path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
				path.EndpointA.ChannelConfig.PortID = portID

				// finish channel handshake
				err = path.EndpointB.ChanOpenTry()
				suite.Require().NoError(err)
				err = path.EndpointA.ChanOpenAck()
				suite.Require().NoError(err)
				err = path.EndpointB.ChanOpenConfirm()
				suite.Require().NoError(err)

			} else {
				suite.Require().Error(err)
			}
		})
	}
}
