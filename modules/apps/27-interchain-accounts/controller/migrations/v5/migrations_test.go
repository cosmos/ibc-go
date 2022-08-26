package v5_test

import (
	"testing"

	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/stretchr/testify/suite"

	v5 "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/migrations/v5"
	"github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v5/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v5/testing"
	ibcmock "github.com/cosmos/ibc-go/v5/testing/mock"
)

type MigrationsTestSuite struct {
	suite.Suite

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	coordinator *ibctesting.Coordinator
	path        *ibctesting.Path
}

func (suite *MigrationsTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.path.EndpointA.ChannelConfig.PortID = icatypes.PortID
	suite.path.EndpointB.ChannelConfig.PortID = icatypes.PortID
	suite.path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	suite.path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	suite.path.EndpointA.ChannelConfig.Version = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
	suite.path.EndpointB.ChannelConfig.Version = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
}

func (suite *MigrationsTestSuite) SetupPath() error {
	if err := suite.RegisterInterchainAccount(suite.path.EndpointA, ibctesting.TestAccAddress); err != nil {
		return err
	}

	if err := suite.path.EndpointB.ChanOpenTry(); err != nil {
		return err
	}

	if err := suite.path.EndpointA.ChanOpenAck(); err != nil {
		return err
	}

	if err := suite.path.EndpointB.ChanOpenConfirm(); err != nil {
		return err
	}

	return nil
}

func (suite *MigrationsTestSuite) RegisterInterchainAccount(endpoint *ibctesting.Endpoint, owner string) error {
	portID, err := icatypes.NewControllerPortID(owner)
	if err != nil {
		return err
	}

	channelSequence := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(endpoint.Chain.GetContext())

	if err := endpoint.Chain.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, owner, endpoint.ChannelConfig.Version); err != nil {
		return err
	}

	// commit state changes for proof verification
	endpoint.Chain.NextBlock()

	// update port/channel ids
	endpoint.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
	endpoint.ChannelConfig.PortID = portID

	return nil
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationsTestSuite))
}

func (suite *MigrationsTestSuite) TestMigrateChannelCapability() {
	suite.SetupTest()

	suite.coordinator.SetupConnections(suite.path)

	err := suite.SetupPath()
	suite.Require().NoError(err)

	capName := host.ChannelCapabilityPath(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)

	// assert the capability is owned by the mock module
	cap, found := suite.chainA.GetSimApp().ScopedICAMockKeeper.GetCapability(suite.chainA.GetContext(), capName)
	suite.Require().NotNil(cap)
	suite.Require().True(found)

	cap, found = suite.chainA.GetSimApp().ScopedICAControllerKeeper.GetCapability(suite.chainA.GetContext(), capName)
	suite.Require().Nil(cap)
	suite.Require().False(found)

	err = v5.MigrateICS27ChannelCapability(
		suite.chainA.GetContext(),
		suite.chainA.GetSimApp().GetMemKey(capabilitytypes.MemStoreKey),
		suite.chainA.GetSimApp().CapabilityKeeper,
		ibcmock.ModuleName+types.SubModuleName,
	)

	suite.Require().NoError(err)

	// assert the capability is now owned by the ICS27 controller submodule
	cap, found = suite.chainA.GetSimApp().ScopedICAControllerKeeper.GetCapability(suite.chainA.GetContext(), capName)
	suite.Require().NotNil(cap)
	suite.Require().True(found)

	cap, found = suite.chainA.GetSimApp().ScopedICAMockKeeper.GetCapability(suite.chainA.GetContext(), capName)
	suite.Require().Nil(cap)
	suite.Require().False(found)
}
