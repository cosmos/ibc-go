package child_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/child"
	childtypes "github.com/cosmos/ibc-go/modules/apps/ccv/child/types"
	parenttypes "github.com/cosmos/ibc-go/modules/apps/ccv/parent/types"
	"github.com/cosmos/ibc-go/modules/apps/ccv/types"
	ccv "github.com/cosmos/ibc-go/modules/apps/ccv/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	ibctmtypes "github.com/cosmos/ibc-go/modules/light-clients/07-tendermint/types"
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/stretchr/testify/suite"
)

type ChildTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains
	parentChain *ibctesting.TestChain
	childChain  *ibctesting.TestChain

	parentClient    *ibctmtypes.ClientState
	parentConsState *ibctmtypes.ConsensusState

	path *ibctesting.Path

	ctx sdk.Context
}

func (suite *ChildTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.parentChain = suite.coordinator.GetChain(ibctesting.GetChainID(0))
	suite.childChain = suite.coordinator.GetChain(ibctesting.GetChainID(1))

	tmConfig := ibctesting.NewTendermintConfig()

	// commit a block on parent chain before creating client
	suite.coordinator.CommitBlock(suite.parentChain)

	// create client and consensus state of parent chain to initialize child chain genesis.
	height := suite.parentChain.LastHeader.GetHeight().(clienttypes.Height)
	UpgradePath := []string{"upgrade", "upgradedIBCState"}

	suite.parentClient = ibctmtypes.NewClientState(
		suite.parentChain.ChainID, tmConfig.TrustLevel, tmConfig.TrustingPeriod, tmConfig.UnbondingPeriod, tmConfig.MaxClockDrift,
		height, commitmenttypes.GetSDKSpecs(), UpgradePath, tmConfig.AllowUpdateAfterExpiry, tmConfig.AllowUpdateAfterMisbehaviour,
	)
	suite.parentConsState = suite.parentChain.LastHeader.ConsensusState()

	childGenesis := types.NewInitialChildGenesisState(suite.parentClient, suite.parentConsState)
	suite.childChain.GetSimApp().ChildKeeper.InitGenesis(suite.childChain.GetContext(), childGenesis)

	// create the ccv path and set child's clientID to genesis client
	path := ibctesting.NewPath(suite.childChain, suite.parentChain)
	path.EndpointA.ChannelConfig.PortID = childtypes.PortID
	path.EndpointB.ChannelConfig.PortID = parenttypes.PortID
	path.EndpointA.ChannelConfig.Version = types.Version
	path.EndpointB.ChannelConfig.Version = types.Version
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	parentClient, ok := suite.childChain.GetSimApp().ChildKeeper.GetParentClient(suite.childChain.GetContext())
	if !ok {
		panic("must already have parent client on child chain")
	}
	// set child endpoint's clientID
	path.EndpointA.ClientID = parentClient

	// create child client on parent chain
	path.EndpointB.CreateClient()

	suite.coordinator.CreateConnections(path)

	suite.ctx = suite.childChain.GetContext()

	suite.path = path
}

func (suite *ChildTestSuite) TestOnChanOpenInit() {
	channelID := "channel-1"
	testCases := []struct {
		name     string
		setup    func(suite *ChildTestSuite)
		expError bool
	}{
		{
			name: "success",
			setup: func(suite *ChildTestSuite) {
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: false,
		},
		{
			name: "invalid: parent channel already established",
			setup: func(suite *ChildTestSuite) {
				suite.childChain.GetSimApp().ChildKeeper.SetParentChannel(suite.ctx, "channel-2")
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: true,
		},
		{
			name: "invalid: UNORDERED channel",
			setup: func(suite *ChildTestSuite) {
				// set path ORDER to UNORDERED
				suite.path.EndpointA.ChannelConfig.Order = channeltypes.UNORDERED
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.UNORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: true,
		},
		{
			name: "invalid: incorrect port",
			setup: func(suite *ChildTestSuite) {
				// set path port to invalid portID
				suite.path.EndpointA.ChannelConfig.PortID = "invalidPort"
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.UNORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: true,
		},
		{
			name: "invalid: incorrect version",
			setup: func(suite *ChildTestSuite) {
				// set path port to invalid version
				suite.path.EndpointA.ChannelConfig.Version = "invalidVersion"
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.UNORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: true,
		},
		{
			name: "invalid: verify parent chain failed",
			setup: func(suite *ChildTestSuite) {
				// setup a new path with parent client on child chain being different from genesis client
				path := ibctesting.NewPath(suite.childChain, suite.parentChain)
				path.EndpointA.ChannelConfig.PortID = childtypes.PortID
				path.EndpointB.ChannelConfig.PortID = parenttypes.PortID
				path.EndpointA.ChannelConfig.Version = types.Version
				path.EndpointB.ChannelConfig.Version = types.Version
				path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
				path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED

				// create child client on parent chain, and parent client on child chain
				path.EndpointB.CreateClient()
				path.EndpointA.CreateClient()

				suite.coordinator.CreateConnections(path)
				suite.path = path

				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case: %s", tc.name), func() {
			suite.SetupTest() // reset suite
			tc.setup(suite)

			childModule := child.NewAppModule(suite.childChain.GetSimApp().ChildKeeper)
			chanCap, err := suite.childChain.App.GetScopedIBCKeeper().NewCapability(suite.ctx, host.ChannelCapabilityPath(childtypes.PortID, suite.path.EndpointA.ChannelID))
			suite.Require().NoError(err)

			err = childModule.OnChanOpenInit(suite.ctx, suite.path.EndpointA.ChannelConfig.Order, []string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID, chanCap, channeltypes.NewCounterparty(parenttypes.PortID, ""), suite.path.EndpointA.ChannelConfig.Version)

			if tc.expError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *ChildTestSuite) TestOnChanOpenTry() {
	// OnOpenTry must error even with correct arguments
	childModule := child.NewAppModule(suite.childChain.GetSimApp().ChildKeeper)
	chanCap, err := suite.childChain.App.GetScopedIBCKeeper().NewCapability(suite.ctx, host.ChannelCapabilityPath(childtypes.PortID, suite.path.EndpointA.ChannelID))
	suite.Require().NoError(err)

	err = childModule.OnChanOpenTry(suite.ctx, channeltypes.ORDERED, []string{"connection-1"}, childtypes.PortID, "channel-1", chanCap,
		channeltypes.NewCounterparty(parenttypes.PortID, "channel-1"), ccv.Version, ccv.Version)
	suite.Require().Error(err, "OnChanOpenTry callback must error on child chain")
}

func (suite *ChildTestSuite) TestOnChanOpenAck() {
	channelID := "channel-1"
	testCases := []struct {
		name     string
		setup    func(suite *ChildTestSuite)
		expError bool
	}{
		{
			name: "success",
			setup: func(suite *ChildTestSuite) {
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: false,
		},
		{
			name: "invalid: parent channel already established",
			setup: func(suite *ChildTestSuite) {
				suite.childChain.GetSimApp().ChildKeeper.SetParentChannel(suite.ctx, "channel-2")
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: true,
		},
		{
			name: "invalid: mismatched versions",
			setup: func(suite *ChildTestSuite) {
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
				// set parent version to invalid version
				suite.path.EndpointB.ChannelConfig.Version = "invalidVersion"
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case: %s", tc.name), func() {
			suite.SetupTest() // reset suite
			tc.setup(suite)

			childModule := child.NewAppModule(suite.childChain.GetSimApp().ChildKeeper)

			err := childModule.OnChanOpenAck(suite.ctx, childtypes.PortID, channelID, suite.path.EndpointB.ChannelConfig.Version)

			if tc.expError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *ChildTestSuite) TestOnChanOpenConfirm() {
	suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, "channel-1",
		channeltypes.NewChannel(
			channeltypes.TRYOPEN, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, "channel-1"),
			[]string{"connection-1"}, ccv.Version,
		))

	childModule := child.NewAppModule(suite.childChain.GetSimApp().ChildKeeper)

	err := childModule.OnChanOpenConfirm(suite.ctx, childtypes.PortID, "channel-1")
	suite.Require().Error(err, "OnChanOpenConfirm must always fail")
}

func (suite *ChildTestSuite) TestOnChanCloseInit() {
	channelID := "channel-1"
	testCases := []struct {
		name     string
		setup    func(suite *ChildTestSuite)
		expError bool
	}{
		{
			name: "can close duplicate in-progress channel once parent channel is established",
			setup: func(suite *ChildTestSuite) {
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
				suite.childChain.GetSimApp().ChildKeeper.SetParentChannel(suite.ctx, "different-channel")
			},
			expError: false,
		},
		{
			name: "can close duplicate open channel once parent channel is established",
			setup: func(suite *ChildTestSuite) {
				// create open channel
				suite.coordinator.CreateChannels(suite.path)
				suite.childChain.GetSimApp().ChildKeeper.SetParentChannel(suite.ctx, "different-channel")
			},
			expError: false,
		},
		{
			name: "cannot close in-progress channel, no established channel yet",
			setup: func(suite *ChildTestSuite) {
				// Set INIT channel on child chain
				suite.childChain.App.GetIBCKeeper().ChannelKeeper.SetChannel(suite.ctx, childtypes.PortID, channelID,
					channeltypes.NewChannel(
						channeltypes.INIT, channeltypes.ORDERED, channeltypes.NewCounterparty(parenttypes.PortID, ""),
						[]string{suite.path.EndpointA.ConnectionID}, suite.path.EndpointA.ChannelConfig.Version),
				)
				suite.path.EndpointA.ChannelID = channelID
			},
			expError: true,
		},
		{
			name: "cannot close parent channel",
			setup: func(suite *ChildTestSuite) {
				// create open channel
				suite.coordinator.CreateChannels(suite.path)
				suite.childChain.GetSimApp().ChildKeeper.SetParentChannel(suite.ctx, suite.path.EndpointA.ChannelID)
			},
			expError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(fmt.Sprintf("Case: %s", tc.name), func() {
			suite.SetupTest() // reset suite
			tc.setup(suite)

			childModule := child.NewAppModule(suite.childChain.GetSimApp().ChildKeeper)

			err := childModule.OnChanCloseInit(suite.ctx, childtypes.PortID, suite.path.EndpointA.ChannelID)

			if tc.expError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func TestChildTestSuite(t *testing.T) {
	suite.Run(t, new(ChildTestSuite))
}
