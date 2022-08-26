package keeper_test

import (
	"github.com/cosmos/ibc-go/v5/modules/apps/icq/types"
)

func (suite *KeeperTestSuite) TestInitGenesis() {
	suite.SetupTest()

	genesisState := types.GenesisState{
		HostPort: TestPort,
		Params: types.Params{
			HostEnabled: false,
			AllowQueries: []string{
				"path/to/query1",
				"path/to/query2",
			},
		},
	}

	suite.chainA.GetSimApp().ICQKeeper.InitGenesis(suite.chainA.GetContext(), genesisState)

	port := suite.chainA.GetSimApp().ICQKeeper.GetPort(suite.chainA.GetContext())
	suite.Require().Equal(TestPort, port)

	expParams := types.NewParams(
		false,
		[]string{
			"path/to/query1",
			"path/to/query2",
		},
	)
	params := suite.chainA.GetSimApp().ICQKeeper.GetParams(suite.chainA.GetContext())
	suite.Require().Equal(expParams, params)
}

func (suite *KeeperTestSuite) TestExportGenesis() {
	suite.SetupTest()

	genesisState := suite.chainA.GetSimApp().ICQKeeper.ExportGenesis(suite.chainA.GetContext())

	suite.Require().Equal(types.PortID, genesisState.GetHostPort())

	expParams := types.DefaultParams()
	suite.Require().Equal(expParams, genesisState.GetParams())
}
