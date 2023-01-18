package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

func (suite *KeeperTestSuite) TestMigratorMigrateTraces() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedTraces transfertypes.Traces
	}{
		{
			"success: two slashes in base denom",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "pool/1", Path: "transfer/channel-0/gamm",
					})
			},
			transfertypes.Traces{
				{
					BaseDenom: "gamm/pool/1", Path: "transfer/channel-0",
				},
			},
		},
		{
			"success: one slash in base denom",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", Path: "transfer/channel-149/erc",
					})
			},
			transfertypes.Traces{
				{
					BaseDenom: "erc/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", Path: "transfer/channel-149",
				},
			},
		},
		{
			"success: multiple slashes in a row in base denom",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "1", Path: "transfer/channel-5/gamm//pool",
					})
			},
			transfertypes.Traces{
				{
					BaseDenom: "gamm//pool/1", Path: "transfer/channel-5",
				},
			},
		},
		{
			"success: multihop base denom",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "transfer/channel-1/uatom", Path: "transfer/channel-0",
					})
			},
			transfertypes.Traces{
				{
					BaseDenom: "uatom", Path: "transfer/channel-0/transfer/channel-1",
				},
			},
		},
		{
			"success: non-standard port",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					transfertypes.DenomTrace{
						BaseDenom: "customport/channel-7/uatom", Path: "transfer/channel-0/transfer/channel-1",
					})
			},
			transfertypes.Traces{
				{
					BaseDenom: "uatom", Path: "transfer/channel-0/transfer/channel-1/customport/channel-7",
				},
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate() // explicitly set up denom traces

			migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
			err := migrator.MigrateTraces(suite.chainA.GetContext())
			suite.Require().NoError(err)

			traces := suite.chainA.GetSimApp().TransferKeeper.GetAllDenomTraces(suite.chainA.GetContext())
			suite.Require().Equal(tc.expectedTraces, traces)
		})
	}
}

func (suite *KeeperTestSuite) TestMigratorMigrateTracesCorruptionDetection() {
	// IBCDenom() previously would return "customport/channel-0/uatom", but now should return ibc/{hash}
	corruptedDenomTrace := transfertypes.DenomTrace{
		BaseDenom: "customport/channel-0/uatom",
		Path:      "",
	}
	suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(suite.chainA.GetContext(), corruptedDenomTrace)

	migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
	suite.Panics(func() {
		migrator.MigrateTraces(suite.chainA.GetContext()) //nolint:errcheck // we shouldn't check the error here because we want to ensure that a panic occurs.
	})
}

func (suite *KeeperTestSuite) TestMigrateTotalEscrowOut() {
	testCases := []struct {
		msg          string
		malleate     func()
		expectedCoin sdk.Coin
	}{
		{
			msg: "Success: account contains native denom",
			malleate: func() {
				suite.chainA.GetSimApp().TransferKeeper.SetIBCOutDenomAmount(
					suite.chainA.GetContext(),
					sdk.DefaultBondDenom,
					sdk.NewInt(100_000_000),
				)
			},
			expectedCoin: sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100_000_000)),
		},
		{
			msg: "failure: amount doesnot contain native denom",
			malleate: func() {
				suite.chainA.GetSimApp().TransferKeeper.SetIBCOutDenomAmount(
					suite.chainA.GetContext(),
					"ibc/stake",
					sdk.NewInt(100_000_000),
				)
			},
			expectedCoin: sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()),
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate() // explicitly set up denom traces

			migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
			migrator.MigrateTotalEscrowOut(suite.chainA.GetContext())

			// check if the migration amount matches the expected amount
			amount := suite.chainA.GetSimApp().TransferKeeper.GetIBCOutDenomAmount(suite.chainA.GetContext(), sdk.DefaultBondDenom)
			suite.Require().Equal(amount, tc.expectedCoin.Amount)
		})
	}
}
