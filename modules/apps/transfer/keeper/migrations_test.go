package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	internaltransfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/types"
	transferkeeper "github.com/cosmos/ibc-go/v9/modules/apps/transfer/keeper"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

func (suite *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedParams transfertypes.Params
	}{
		{
			"success: default params",
			func() {
				params := transfertypes.DefaultParams()
				subspace := suite.chainA.GetSimApp().GetSubspace(transfertypes.ModuleName)
				subspace.SetParamSet(suite.chainA.GetContext(), &params) // set params
			},
			transfertypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("case %s", tc.msg), func() {
			suite.SetupTest() // reset

			tc.malleate() // explicitly set params

			migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
			err := migrator.MigrateParams(suite.chainA.GetContext())
			suite.Require().NoError(err)

			params := suite.chainA.GetSimApp().TransferKeeper.GetParams(suite.chainA.GetContext())
			suite.Require().Equal(tc.expectedParams, params)
		})
	}
}

func (suite *KeeperTestSuite) TestMigratorMigrateDenomTraceToDenom() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedDenoms transfertypes.Denoms
	}{
		{
			"success: no trace",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "uatom",
					})
			},
			transfertypes.Denoms{
				transfertypes.NewDenom("uatom"),
			},
		},
		{
			"success: single trace",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "uatom", Path: "transfer/channel-49",
					})
			},
			transfertypes.Denoms{
				transfertypes.NewDenom("uatom", transfertypes.NewHop("transfer", "channel-49")),
			},
		},
		{
			"success: multiple trace",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "uatom", Path: "transfer/channel-49/transfer/channel-32/transfer/channel-2",
					})
			},
			transfertypes.Denoms{
				transfertypes.NewDenom("uatom", transfertypes.NewHop("transfer", "channel-49"), transfertypes.NewHop("transfer", "channel-32"), transfertypes.NewHop("transfer", "channel-2")),
			},
		},
		{
			"success: many denoms",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "uatom", Path: "transfer/channel-49",
					})
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "pineapple", Path: "transfer/channel-0",
					})
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "apple", Path: "transfer/channel-0",
					})
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "cucumber", Path: "transfer/channel-102/transfer/channel-0",
					})
			},
			transfertypes.Denoms{
				transfertypes.NewDenom("apple", transfertypes.NewHop("transfer", "channel-0")),
				transfertypes.NewDenom("cucumber", transfertypes.NewHop("transfer", "channel-102"), transfertypes.NewHop("transfer", "channel-0")),
				transfertypes.NewDenom("pineapple", transfertypes.NewHop("transfer", "channel-0")),
				transfertypes.NewDenom("uatom", transfertypes.NewHop("transfer", "channel-49")),
			},
		},

		{
			"success: two slashes in base denom",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "gamm/pool/1", Path: "transfer/channel-0",
					})
			},
			transfertypes.Denoms{
				transfertypes.NewDenom("gamm/pool/1", transfertypes.NewHop("transfer", "channel-0")),
			},
		},
		{
			"success: one slash in base denom",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "erc/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", Path: "transfer/channel-149",
					})
			},
			transfertypes.Denoms{
				transfertypes.NewDenom("erc/0x85bcBCd7e79Ec36f4fBBDc54F90C643d921151AA", transfertypes.NewHop("transfer", "channel-149")),
			},
		},
		{
			"success: non-standard port",
			func() {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					suite.chainA.GetContext(),
					internaltransfertypes.DenomTrace{
						BaseDenom: "uatom", Path: "transfer/channel-0/customport/channel-7",
					})
			},
			transfertypes.Denoms{
				transfertypes.NewDenom("uatom", transfertypes.NewHop("transfer", "channel-0"), transfertypes.NewHop("customport", "channel-7")),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			tc.malleate()

			migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
			err := migrator.MigrateDenomTraceToDenom(suite.chainA.GetContext())
			suite.Require().NoError(err)

			denoms := suite.chainA.GetSimApp().TransferKeeper.GetAllDenoms(suite.chainA.GetContext())
			suite.Require().Equal(tc.expectedDenoms, denoms)

			// assert no leftover denom traces
			suite.chainA.GetSimApp().TransferKeeper.IterateDenomTraces(suite.chainA.GetContext(),
				func(dt internaltransfertypes.DenomTrace) (stop bool) {
					suite.FailNow("DenomTrace key still exists", dt)
					return false
				},
			)
		})
	}
}

func (suite *KeeperTestSuite) TestMigratorMigrateDenomTraceToDenomCorruptionDetection() {
	testCases := []struct {
		name       string
		denomTrace internaltransfertypes.DenomTrace
	}{
		{
			"corrupted denom trace, denom.IBCHash() does not match",
			internaltransfertypes.DenomTrace{
				BaseDenom: "customport/channel-0/uatom",
				Path:      "",
			},
		},
		{
			"invalid denom trace, base denom is empty",
			internaltransfertypes.DenomTrace{
				BaseDenom: "",
				Path:      "transfer/channel-0",
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(suite.chainA.GetContext(), tc.denomTrace)

			migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
			suite.Panics(func() {
				migrator.MigrateDenomTraceToDenom(suite.chainA.GetContext()) //nolint:errcheck // we shouldn't check the error here because we want to ensure that a panic occurs.
			})
		})
	}
}

func (suite *KeeperTestSuite) TestMigrateTotalEscrowForDenom() {
	var (
		path  *ibctesting.Path
		denom string
	)

	testCases := []struct {
		msg               string
		malleate          func()
		expectedEscrowAmt sdkmath.Int
	}{
		{
			"success: one native denom escrowed in one channel",
			func() {
				denom = sdk.DefaultBondDenom
				escrowAddress := transfertypes.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				coin := ibctesting.TestCoin

				// funds the escrow account to have balance
				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrowAddress, sdk.NewCoins(coin)))
			},
			sdkmath.NewInt(100),
		},
		{
			"success: one native denom escrowed in two channels",
			func() {
				denom = sdk.DefaultBondDenom
				extraPath := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
				extraPath.Setup()

				escrowAddress1 := transfertypes.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				escrowAddress2 := transfertypes.GetEscrowAddress(extraPath.EndpointA.ChannelConfig.PortID, extraPath.EndpointA.ChannelID)
				coin1 := ibctesting.TestCoin
				coin2 := ibctesting.TestCoin

				// funds the escrow accounts to have balance
				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrowAddress1, sdk.NewCoins(coin1)))
				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrowAddress2, sdk.NewCoins(coin2)))
			},
			sdkmath.NewInt(200),
		},
		{
			"success: valid ibc denom escrowed in one channel",
			func() {
				escrowAddress := transfertypes.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				voucherDenom := transfertypes.NewDenom(sdk.DefaultBondDenom, transfertypes.NewHop(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
				coin := sdk.NewCoin(voucherDenom.IBCDenom(), sdkmath.NewInt(100))
				denom = voucherDenom.IBCDenom()

				// funds the escrow account to have balance
				suite.Require().NoError(banktestutil.FundAccount(suite.chainA.GetContext(), suite.chainA.GetSimApp().BankKeeper, escrowAddress, sdk.NewCoins(coin)))
			},
			sdkmath.NewInt(100),
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset

			path = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
			path.Setup()

			tc.malleate() // explicitly fund escrow account

			migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
			suite.Require().NoError(migrator.MigrateTotalEscrowForDenom(suite.chainA.GetContext()))

			// check that the migration set the expected amount for both native and IBC tokens
			amount := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), denom)
			suite.Require().Equal(tc.expectedEscrowAmt, amount.Amount)
		})
	}
}

func (suite *KeeperTestSuite) TestMigratorMigrateMetadata() {
	var (
		denomTraces      []internaltransfertypes.DenomTrace
		expectedMetadata []banktypes.Metadata
	)

	testCases := []struct {
		msg      string
		malleate func()
	}{
		{
			"success with one denom trace with one hop",
			func() {
				denomTraces = []internaltransfertypes.DenomTrace{
					{
						BaseDenom: "foo",
						Path:      "transfer/channel-0",
					},
				}

				expectedMetadata = []banktypes.Metadata{
					{
						Description: "IBC token from transfer/channel-0/foo",
						DenomUnits: []*banktypes.DenomUnit{
							{
								Denom:    "foo",
								Exponent: 0,
							},
						},
						Base:    denomTraces[0].IBCDenom(), // ibc/EB7094899ACFB7A6F2A67DB084DEE2E9A83DEFAA5DEF92D9A9814FFD9FF673FA
						Display: "transfer/channel-0/foo",
						Name:    "transfer/channel-0/foo IBC token",
						Symbol:  "FOO",
					},
				}
			},
		},
		{
			"success with one denom trace with two hops",
			func() {
				denomTraces = []internaltransfertypes.DenomTrace{
					{
						BaseDenom: "ubar",
						Path:      "transfer/channel-1/transfer/channel-2",
					},
				}

				expectedMetadata = []banktypes.Metadata{
					{
						Description: "IBC token from transfer/channel-1/transfer/channel-2/ubar",
						DenomUnits: []*banktypes.DenomUnit{
							{
								Denom:    "ubar",
								Exponent: 0,
							},
						},
						Base:    denomTraces[0].IBCDenom(), // ibc/8243B3EAA19BAB1DB3B0020B81C0C5A953E7B22C042CEE44E639A11A238BA57C
						Display: "transfer/channel-1/transfer/channel-2/ubar",
						Name:    "transfer/channel-1/transfer/channel-2/ubar IBC token",
						Symbol:  "UBAR",
					},
				}
			},
		},
		{
			"success with two denom traces with one hop",
			func() {
				denomTraces = []internaltransfertypes.DenomTrace{
					{
						BaseDenom: "foo",
						Path:      "transfer/channel-0",
					},
					{
						BaseDenom: "bar",
						Path:      "transfer/channel-0",
					},
				}

				expectedMetadata = []banktypes.Metadata{
					{
						Description: "IBC token from transfer/channel-0/foo",
						DenomUnits: []*banktypes.DenomUnit{
							{
								Denom:    "foo",
								Exponent: 0,
							},
						},
						Base:    denomTraces[0].IBCDenom(), // ibc/EB7094899ACFB7A6F2A67DB084DEE2E9A83DEFAA5DEF92D9A9814FFD9FF673FA
						Display: "transfer/channel-0/foo",
						Name:    "transfer/channel-0/foo IBC token",
						Symbol:  "FOO",
					},
					{
						Description: "IBC token from transfer/channel-0/bar",
						DenomUnits: []*banktypes.DenomUnit{
							{
								Denom:    "bar",
								Exponent: 0,
							},
						},
						Base:    denomTraces[1].IBCDenom(), // ibc/E1530E21F1848B6C29C9E89256D43E294976897611A61741CACBA55BE21736F5
						Display: "transfer/channel-0/bar",
						Name:    "transfer/channel-0/bar IBC token",
						Symbol:  "BAR",
					},
				}
			},
		},
		{
			"success with two denom traces, metadata for one of them already exists",
			func() {
				denomTraces = []internaltransfertypes.DenomTrace{
					{
						BaseDenom: "foo",
						Path:      "transfer/channel-0",
					},
					{
						BaseDenom: "bar",
						Path:      "transfer/channel-0",
					},
				}

				expectedMetadata = []banktypes.Metadata{
					{
						Description: "IBC token from transfer/channel-0/foo",
						DenomUnits: []*banktypes.DenomUnit{
							{
								Denom:    "foo",
								Exponent: 0,
							},
						},
						Base:    denomTraces[0].IBCDenom(), // ibc/EB7094899ACFB7A6F2A67DB084DEE2E9A83DEFAA5DEF92D9A9814FFD9FF673FA
						Display: "transfer/channel-0/foo",
						Name:    "transfer/channel-0/foo IBC token",
						Symbol:  "FOO",
					},
					{
						Description: "IBC token from transfer/channel-0/bar",
						DenomUnits: []*banktypes.DenomUnit{
							{
								Denom:    "bar",
								Exponent: 0,
							},
						},
						Base:    denomTraces[1].IBCDenom(), // ibc/E1530E21F1848B6C29C9E89256D43E294976897611A61741CACBA55BE21736F5
						Display: "transfer/channel-0/bar",
						Name:    "transfer/channel-0/bar IBC token",
						Symbol:  "BAR",
					},
				}

				// set metadata for one of the tokens, so that it exists already in state before doing the migration
				suite.chainA.GetSimApp().BankKeeper.SetDenomMetaData(suite.chainA.GetContext(), expectedMetadata[1])
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			suite.SetupTest() // reset
			ctx := suite.chainA.GetContext()

			tc.malleate()

			for _, denomTrace := range denomTraces {
				suite.chainA.GetSimApp().TransferKeeper.SetDenomTrace(ctx, denomTrace)
			}

			// run migration
			migrator := transferkeeper.NewMigrator(suite.chainA.GetSimApp().TransferKeeper)
			err := migrator.MigrateDenomMetadata(ctx)
			suite.Require().NoError(err)

			for _, expMetadata := range expectedMetadata {
				denomMetadata, found := suite.chainA.GetSimApp().BankKeeper.GetDenomMetaData(ctx, expMetadata.Base)
				suite.Require().True(found)
				suite.Require().Equal(expMetadata, denomMetadata)
			}
		})
	}
}
