package fee_test

import (
	"fmt"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
)

func (suite *FeeTestSuite) TestOnChanOpenInit() {
	testCases := []struct {
		name    string
		version string
		expPass bool
	}{
		{
			"valid fee middleware and transfer version",
			"fee29-1:ics20-1",
			true,
		},
		{
			"invalid fee middleware version",
			"otherfee28-1:ics20-1",
			false,
		},
		{
			"invalid transfer version",
			"fee29-1:wrongics20-1",
			false,
		},
		{
			"incorrect wrapping delimiter",
			"fee29-1//ics20-1",
			false,
		},
		{
			"transfer version not wrapped",
			"fee29-1",
			false,
		},
		{
			"fee version not included",
			"ics20-1",
			false,
		},
		{
			"hanging delimiter",
			"fee29-1:ics20-1:",
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()

			ctx := suite.chainA.GetContext()
			cap, err := suite.chainA.GetSimApp().ScopedIBCKeeper.NewCapability(ctx, host.ChannelCapabilityPath(transfertypes.FeePortID, "channel-1"))
			suite.Require().NoError(err)
			err = suite.moduleA.OnChanOpenInit(
				ctx,
				channeltypes.UNORDERED,
				[]string{"connection-1"},
				transfertypes.FeePortID,
				"channel-1",
				cap,
				channeltypes.NewCounterparty(transfertypes.FeePortID, ""),
				tc.version,
			)

			if tc.expPass {
				suite.Require().NoError(err, "unexpected error from version: %s", tc.version)

				// check that capabilities are properly claimed and issued
				ctx := suite.chainA.GetContext()
				ibcCap, ok := suite.chainA.GetSimApp().ScopedIBCFeeKeeper.GetCapability(ctx, host.ChannelCapabilityPath(transfertypes.FeePortID, "channel-1"))
				suite.Require().NotNil(ibcCap, "IBC capability is nil on fee keeper")
				suite.Require().True(ok)

				appFeeCap, ok := suite.chainA.GetSimApp().ScopedIBCFeeKeeper.GetCapability(ctx, types.AppCapabilityName(transfertypes.FeePortID, "channel-1"))
				suite.Require().NotNil(appFeeCap, "App capability not created or owned by ibc fee keeper")
				suite.Require().True(ok)
				transferCap, ok := suite.chainA.GetSimApp().ScopedTransferKeeper.GetCapability(ctx, host.ChannelCapabilityPath(transfertypes.FeePortID, "channel-1"))
				suite.Require().NotNil(transferCap, "App capability not claimed by transfer keeper")
				suite.Require().True(ok)
				suite.Require().Equal(appFeeCap, transferCap, "app capabilities not equal")
			} else {
				suite.Require().Error(err, "error not returned for version: %s", tc.version)
			}
		})
	}
}

func (suite *FeeTestSuite) TestOnChanOpenTry() {
	testCases := []struct {
		name      string
		version   string
		cpVersion string
		capExists bool
		expPass   bool
	}{
		{
			"valid fee middleware and transfer version",
			"fee29-1:ics20-1",
			"fee29-1:ics20-1",
			false,
			true,
		},
		{
			"valid fee middleware and transfer version, crossing hellos",
			"fee29-1:ics20-1",
			"fee29-1:ics20-1",
			true,
			true,
		},
		{
			"invalid fee middleware version",
			"otherfee28-1:ics20-1",
			"fee29-1:ics20-1",
			false,
			false,
		},
		{
			"invalid counterparty fee middleware version",
			"fee29-1:ics20-1",
			"wrongfee29-1:ics20-1",
			false,
			false,
		},
		{
			"invalid transfer version",
			"fee29-1:wrongics20-1",
			"fee29-1:ics20-1",
			false,
			false,
		},
		{
			"invalid counterparty transfer version",
			"fee29-1:ics20-1",
			"fee29-1:wrongics20-1",
			false,
			false,
		},
		{
			"transfer version not wrapped",
			"fee29-1",
			"fee29-1:ics20-1",
			false,
			false,
		},
		{
			"counterparty transfer version not wrapped",
			"fee29-1:ics20-1",
			"fee29-1",
			false,
			false,
		},
		{
			"fee version not included",
			"ics20-1",
			"fee29-1:ics20-1",
			false,
			false,
		},
		{
			"transfer version not included",
			"fee29-1:ics20-1",
			"ics20-1",
			false,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()
			suite.coordinator.SetupClients(suite.path)
			suite.coordinator.SetupConnections(suite.path)
			suite.path.EndpointB.ChanOpenInit()

			if tc.capExists {
				suite.path.EndpointA.ChanOpenInit()
			}

			fmt.Println("port:", suite.path.EndpointA.ChannelConfig.PortID)
			err := suite.path.EndpointA.ChanOpenTry()

			if tc.expPass {
				suite.Require().NoError(err, "unexpected error from version: %s", tc.version)

				// check that capabilities are properly claimed and issued
				ctx := suite.chainA.GetContext()
				ibcCap, ok := suite.chainA.GetSimApp().ScopedIBCFeeKeeper.GetCapability(ctx, host.ChannelCapabilityPath(transfertypes.FeePortID, suite.path.EndpointA.ChannelID))
				suite.Require().NotNil(ibcCap, "IBC capability is nil on fee keeper: %s", host.ChannelCapabilityPath(transfertypes.FeePortID, suite.path.EndpointA.ChannelID))
				suite.Require().True(ok)

				appFeeCap, ok := suite.chainA.GetSimApp().ScopedIBCFeeKeeper.GetCapability(ctx, types.AppCapabilityName(transfertypes.FeePortID, suite.path.EndpointA.ChannelID))
				suite.Require().NotNil(appFeeCap, "App capability not created or owned by ibc fee keeper")
				suite.Require().True(ok)
				transferCap, ok := suite.chainA.GetSimApp().ScopedTransferKeeper.GetCapability(ctx, host.ChannelCapabilityPath(transfertypes.FeePortID, suite.path.EndpointA.ChannelID))
				suite.Require().NotNil(transferCap, "App capability not claimed by transfer keeper")
				suite.Require().True(ok)
				suite.Require().Equal(appFeeCap, transferCap, "app capabilities not equal")
			} else {
				suite.Require().Error(err, "error not returned for version: %s", tc.version)
			}
		})
	}
}

func (suite *FeeTestSuite) TestOnChanOpenAck() {
	testCases := []struct {
		name      string
		cpVersion string
		expPass   bool
	}{
		{
			"success",
			"fee29-1:ics20-1",
			true,
		},
		{
			"invalid fee version",
			"fee29-3:ics20-1",
			false,
		},
		{
			"invalid transfer version",
			"fee29-1:ics20-4",
			false,
		},
	}

	for _, tc := range testCases {
		ctx := suite.chainA.GetContext()
		err := suite.moduleA.OnChanOpenAck(ctx, transfertypes.FeePortID, "channel-1", tc.cpVersion)
		if tc.expPass {
			suite.Require().NoError(err, "unexpected error for case: %s", tc.name)
		} else {
			suite.Require().Error(err, "%s expected error but returned none", tc.name)
		}
	}
}
