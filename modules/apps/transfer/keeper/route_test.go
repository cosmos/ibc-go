package keeper_test

// TestSetChainToTuple tests the SetChainToTuple and GetChainToTuple functions
func (suite *KeeperTestSuite) TestSetChainToTuple() {
	cases := []struct {
		description string
		chainID     string
		channel     string
		port        string
	}{
		{
			"basic test",
			"osmosis-1",
			"channel-0",
			"transfer",
		},
	}

	for _, tc := range cases {
		suite.SetupTest() // reset
		suite.Run(tc.description, func() {
			// set the chain
			suite.chainA.GetSimApp().TransferKeeper.SetChainToTuple(suite.chainA.GetContext(), tc.chainID, tc.channel, tc.port)

			// get the chain
			channel, port, err := suite.chainA.GetSimApp().TransferKeeper.GetChainToTuple(suite.chainA.GetContext(), tc.chainID)
			suite.Require().NoError(err)

			suite.Require().Equal(tc.channel, channel)
			suite.Require().Equal(tc.port, port)
		})
	}
}

// TestSetTupleToChain tests the SetTupleToChain and GetTupleToChain functions
func (suite *KeeperTestSuite) TestSetTupleToChain() {
	cases := []struct {
		description string
		chainID     string
		channel     string
		port        string
	}{
		{
			"basic test",
			"osmosis-1",
			"channel-0",
			"transfer",
		},
	}

	for _, tc := range cases {
		suite.SetupTest() // reset
		suite.Run(tc.description, func() {
			// set the tuple
			suite.chainA.GetSimApp().TransferKeeper.SetTupleToChain(suite.chainA.GetContext(), tc.chainID, tc.channel, tc.port)

			// get the tuple
			chainID, err := suite.chainA.GetSimApp().TransferKeeper.GetTupleToChain(suite.chainA.GetContext(), tc.channel, tc.port)
			suite.Require().NoError(err)

			suite.Require().Equal(tc.chainID, chainID)
		})
	}
}
