package keeper_test

import (
	"fmt"

	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
)

func (suite *KeeperTestSuite) TestMigrator_Migrate1to2() {

	testCases := []struct {
		msg         string
		malleate    func()
		doMigration bool
	}{

		{
			"success: denom traces updated",
			func() {
				//set a multitude of different types of denom traces
				// base denom ending in '/'
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "uatom/", Path: "transfer/channelToA",
					})
				// single '/' in base denom
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", Path: "transfer/channelToA",
					})
				// multiple '/'s in base denom
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "erc20/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", Path: "transfer/channelToA",
					})
				// multiple double '/'s in base denom
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "gamm//pool//1", Path: "transfer/channelToA",
					})
				// multiple port/channel pairs
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "uatom", Path: "transfer/channelToA/transfer/channelToB",
					})
			},
			true,
		},
		{
			"failure",
			func() {
			},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.msg), func() {
			suite.SetupTest() // reset
			tc.malleate()     // explicitly set up denom traces
			// migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
			// err := migrator.Migrate1to2(suite.chainA.GetContext())
			if tc.doMigration {
				// suite.Require().NoError(err)
				traces := suite.chainA.GetSimApp().TransferKeeper.GetAllDenomTraces(suite.chainA.GetContext())
				for _, t := range traces {
					err := t.Validate()
					suite.Require().NoError(err)
				}
			} else {
				// suite.Require().Error(err)
			}
		})
	}
}
