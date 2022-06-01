package keeper_test

import "github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"

func (suite *KeeperTestSuite) TestParams() {
	expParams := types.DefaultParams()

	params := suite.chainA.GetSimApp().IBCFeeKeeper.GetParams(suite.chainA.GetContext())
	suite.Require().Equal(expParams, params)

	expParams.DistributionAddress = suite.chainA.SenderAccount.GetAddress().String()
	suite.chainA.GetSimApp().IBCFeeKeeper.SetParams(suite.chainA.GetContext(), expParams)
	params = suite.chainA.GetSimApp().IBCFeeKeeper.GetParams(suite.chainA.GetContext())
	suite.Require().Equal(expParams, params)
}
