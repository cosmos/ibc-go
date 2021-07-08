package keeper_test

func (suite *KeeperTestSuite) TestInitialGenesis() {
	genesis := suite.childChain.GetSimApp().ChildKeeper.ExportGenesis(suite.childChain.GetContext())

	suite.Require().Equal(suite.parentClient, genesis.ParentClientState)
	suite.Require().Equal(suite.parentConsState, genesis.ParentConsensusState)

	suite.Require().NotPanics(func() {
		suite.childChain.GetSimApp().ChildKeeper.InitGenesis(suite.childChain.GetContext(), genesis)
	})
}
