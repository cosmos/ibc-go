package v6_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

type MigrationsTestSuite struct {
	testifysuite.Suite

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	coordinator *ibctesting.Coordinator
	path        *ibctesting.Path
}

func (suite *MigrationsTestSuite) SetupTest() {
	version := icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)

	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))

	suite.path = ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	suite.path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	suite.path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	suite.path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	suite.path.EndpointA.ChannelConfig.Version = version
	suite.path.EndpointB.ChannelConfig.Version = version
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

	return suite.path.EndpointB.ChanOpenConfirm()
}

func (*MigrationsTestSuite) RegisterInterchainAccount(endpoint *ibctesting.Endpoint, owner string) error {
	portID, err := icatypes.NewControllerPortID(owner)
	if err != nil {
		return err
	}

	channelSequence := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(endpoint.Chain.GetContext())

	if err := endpoint.Chain.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, owner, endpoint.ChannelConfig.Version, channeltypes.ORDERED); err != nil {
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
	testifysuite.Run(t, new(MigrationsTestSuite))
}

// CreateMockCapabilities creates an additional two capabilities used for testing purposes:
// 1. A capability with a single owner
// 2. A capability with two owners, neither of which is "ibc"
func (suite *MigrationsTestSuite) CreateMockCapabilities() {
	capability, err := suite.chainA.GetSimApp().ScopedIBCMockKeeper.NewCapability(suite.chainA.GetContext(), "mock_one")
	suite.Require().NoError(err)
	suite.Require().NotNil(capability)

	capability, err = suite.chainA.GetSimApp().ScopedICAMockKeeper.NewCapability(suite.chainA.GetContext(), "mock_two")
	suite.Require().NoError(err)
	suite.Require().NotNil(capability)

	err = suite.chainA.GetSimApp().ScopedIBCMockKeeper.ClaimCapability(suite.chainA.GetContext(), capability, "mock_two")
	suite.Require().NoError(err)
}

// AssertMockCapabiltiesUnchanged authenticates the mock capabilities created at the start of the test to ensure they remain unchanged
func (suite *MigrationsTestSuite) AssertMockCapabiltiesUnchanged() {
	capability, found := suite.chainA.GetSimApp().ScopedIBCMockKeeper.GetCapability(suite.chainA.GetContext(), "mock_one")
	suite.Require().True(found)
	suite.Require().NotNil(capability)

	capability, found = suite.chainA.GetSimApp().ScopedIBCMockKeeper.GetCapability(suite.chainA.GetContext(), "mock_two")
	suite.Require().True(found)
	suite.Require().NotNil(capability)

	isAuthenticated := suite.chainA.GetSimApp().ScopedICAMockKeeper.AuthenticateCapability(suite.chainA.GetContext(), capability, "mock_two")
	suite.Require().True(isAuthenticated)
}

// ResetMemstore removes all existing fwd and rev capability kv pairs and deletes `KeyMemInitialised` from the x/capability memstore.
// This effectively mocks a new chain binary being started. Migration code is run against persisted state only and allows the memstore to be reinitialised.
func (suite *MigrationsTestSuite) ResetMemStore() {
	memStore := suite.chainA.GetContext().KVStore(suite.chainA.GetSimApp().GetMemKey(capabilitytypes.MemStoreKey))
	memStore.Delete(capabilitytypes.KeyMemInitialized)

	iterator := memStore.Iterator(nil, nil)
	defer sdk.LogDeferred(suite.chainA.GetContext().Logger(), func() error { return iterator.Close() })

	for ; iterator.Valid(); iterator.Next() {
		memStore.Delete(iterator.Key())
	}
}
