package keeper_test

import (
	"fmt"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	
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

func (suite *KeeperTestSuite) TestMigratorMigrateMetadata() {
	DenomTraces := []transfertypes.DenomTrace{
		{
			BaseDenom: "foo",
			Path:      "transfer/channel-0",
		},
		{
			BaseDenom: "ubar",
			Path:      "transfer/channel-1/transfer/channel-2",
		},
	}

	expectedMetaData := []banktypes.Metadata{
		{
			Description: "IBC Token from transfer/channel-0/foo",
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    "foo",
					Exponent: 0,
				},
			},
			Base:    DenomTraces[0].IBCDenom(), // ibc/EB7094899ACFB7A6F2A67DB084DEE2E9A83DEFAA5DEF92D9A9814FFD9FF673FA
			Display: "transfer/channel-0/foo",
			Name:    "transfer/channel-0/foo IBC Token",
			Symbol:  "FOO",
		},
		{
			Description: "IBC Token from transfer/channel-1/transfer/channel-2/ubar",
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    "ubar",
					Exponent: 0,
				},
			},
			Base:    DenomTraces[1].IBCDenom(), // ibc/8243B3EAA19BAB1DB3B0020B81C0C5A953E7B22C042CEE44E639A11A238BA57C
			Display: "transfer/channel-1/transfer/channel-2/ubar",
			Name:    "transfer/channel-1/transfer/channel-2/ubar IBC Token",
			Symbol:  "UBAR",
		},
	}

	ctx := suite.chainA.GetContext()

	// set denom traces
	for _, dt := range DenomTraces {
		suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(ctx, dt)
	}

	// run migration
	migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
	err := migrator.MigrateMetadata(suite.chainA.GetContext())
	suite.Require().NoError(err)

	bk := suite.chainA.GetSimApp().BankKeeper
	for _, exp := range expectedMetaData {
		got, ok := bk.GetDenomMetaData(ctx, exp.Base)
		suite.Require().True(ok)
		suite.Require().Equal(exp, got)
	}
}
