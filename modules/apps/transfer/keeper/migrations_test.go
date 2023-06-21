package keeper_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"

	transferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (s *KeeperTestSuite) TestMigratorMigrateParams() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedParams transfertypes.Params
	}{
		{
			"success: default params",
			func() {
				params := transfertypes.DefaultParams()
				subspace := s.chainA.GetSimApp().GetSubspace(transfertypes.ModuleName)
				subspace.SetParamSet(s.chainA.GetContext(), &params) // set params
			},
			transfertypes.DefaultParams(),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate() // explicitly set params

			migrator := transferkeeper.NewMigrator(s.chainA.GetSimApp().TransferKeeper)
			err := migrator.MigrateParams(s.chainA.GetContext())
			s.Require().NoError(err)

			params := s.chainA.GetSimApp().TransferKeeper.GetParams(s.chainA.GetContext())
			s.Require().Equal(tc.expectedParams, params)
		})
	}
}

func (s *KeeperTestSuite) TestMigratorMigrateTraces() {
	testCases := []struct {
		msg            string
		malleate       func()
		expectedTraces transfertypes.Traces
	}{
		{
			"success: two slashes in base denom",
			func() {
				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					s.chainA.GetContext(),
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
				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					s.chainA.GetContext(),
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
				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					s.chainA.GetContext(),
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
				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					s.chainA.GetContext(),
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
				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(
					s.chainA.GetContext(),
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
		s.Run(fmt.Sprintf("case %s", tc.msg), func() {
			s.SetupTest() // reset

			tc.malleate() // explicitly set up denom traces

			migrator := transferkeeper.NewMigrator(s.chainA.GetSimApp().TransferKeeper)
			err := migrator.MigrateTraces(s.chainA.GetContext())
			s.Require().NoError(err)

			traces := s.chainA.GetSimApp().TransferKeeper.GetAllDenomTraces(s.chainA.GetContext())
			s.Require().Equal(tc.expectedTraces, traces)
		})
	}
}

func (s *KeeperTestSuite) TestMigratorMigrateTracesCorruptionDetection() {
	// IBCDenom() previously would return "customport/channel-0/uatom", but now should return ibc/{hash}
	corruptedDenomTrace := transfertypes.DenomTrace{
		BaseDenom: "customport/channel-0/uatom",
		Path:      "",
	}
	s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(s.chainA.GetContext(), corruptedDenomTrace)

	migrator := transferkeeper.NewMigrator(s.chainA.GetSimApp().TransferKeeper)
	s.Panics(func() {
		migrator.MigrateTraces(s.chainA.GetContext()) //nolint:errcheck // we shouldn't check the error here because we want to ensure that a panic occurs.
	})
}

func (s *KeeperTestSuite) TestMigrateTotalEscrowForDenom() {
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
				coin := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))

				// funds the escrow account to have balance
				s.Require().NoError(banktestutil.FundAccount(s.chainA.GetSimApp().BankKeeper, s.chainA.GetContext(), escrowAddress, sdk.NewCoins(coin)))
			},
			sdkmath.NewInt(100),
		},
		{
			"success: one native denom escrowed in two channels",
			func() {
				denom = sdk.DefaultBondDenom
				extraPath := NewTransferPath(s.chainA, s.chainB)
				s.coordinator.Setup(extraPath)

				escrowAddress1 := transfertypes.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				escrowAddress2 := transfertypes.GetEscrowAddress(extraPath.EndpointA.ChannelConfig.PortID, extraPath.EndpointA.ChannelID)
				coin1 := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))
				coin2 := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))

				// funds the escrow accounts to have balance
				s.Require().NoError(banktestutil.FundAccount(s.chainA.GetSimApp().BankKeeper, s.chainA.GetContext(), escrowAddress1, sdk.NewCoins(coin1)))
				s.Require().NoError(banktestutil.FundAccount(s.chainA.GetSimApp().BankKeeper, s.chainA.GetContext(), escrowAddress2, sdk.NewCoins(coin2)))
			},
			sdkmath.NewInt(200),
		},
		{
			"success: valid ibc denom escrowed in one channel",
			func() {
				escrowAddress := transfertypes.GetEscrowAddress(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
				trace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, sdk.DefaultBondDenom))
				coin := sdk.NewCoin(trace.IBCDenom(), sdkmath.NewInt(100))
				denom = trace.IBCDenom()

				s.chainA.GetSimApp().TransferKeeper.SetDenomTrace(s.chainA.GetContext(), trace)

				// funds the escrow account to have balance
				s.Require().NoError(banktestutil.FundAccount(s.chainA.GetSimApp().BankKeeper, s.chainA.GetContext(), escrowAddress, sdk.NewCoins(coin)))
			},
			sdkmath.NewInt(100),
		},
	}

	for _, tc := range testCases {
		s.Run(fmt.Sprintf("Case %s", tc.msg), func() {
			s.SetupTest() // reset

			path = NewTransferPath(s.chainA, s.chainB)
			s.coordinator.Setup(path)

			tc.malleate() // explicitly fund escrow account

			migrator := transferkeeper.NewMigrator(s.chainA.GetSimApp().TransferKeeper)
			s.Require().NoError(migrator.MigrateTotalEscrowForDenom(s.chainA.GetContext()))

			// check that the migration set the expected amount for both native and IBC tokens
			amount := s.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(s.chainA.GetContext(), denom)
			s.Require().Equal(tc.expectedEscrowAmt, amount.Amount)
		})
	}
}
