package keeper_test

import (
	ibctesting "github.com/cosmos/ibc-go/testing"
)

func (suite *KeeperTestSuite) TestTrySendTx() {
	var (
		path *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		//		{
		//			"active channel not found", func() {
		//				// need to ensure no active channel set
		//			}, false,
		//		},
		//		{
		//			"channel not found", func() {
		//				// need to close the channel
		//			}, false,
		//		},
		//		{
		//			"invalid packet data", func() {
		//				// should fail when passing something other than sdk.Msg
		//			}, false,
		//		},
		//		{
		//			"module does not own channel capability", func() {
		//				// should fail if module capability not ok
		//			}, false,
		//		},
		//		{
		//			"next sequence not found", func() {
		//				// should fail when next sequence is not found
		//			}, false,
		//		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest() // reset
			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := InitInterchainAccount(path.EndpointA, "owner")
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			suite.Require().NoError(err)

			err = path.EndpointA.ChanOpenAck()
			suite.Require().NoError(err)

			err = path.EndpointB.ChanOpenConfirm()
			suite.Require().NoError(err)

			// ensure counterparty is up to date
			path.EndpointA.UpdateClient()

			tc.malleate() // explicitly change fields in channel and testChannel

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}
