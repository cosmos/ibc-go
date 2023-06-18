package v6_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/stretchr/testify/suite"

	v6 "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/migrations/v6"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

type MigrationsTestSuite struct {
	suite.Suite

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain

	coordinator *ibctesting.Coordinator
	path        *ibctesting.Path
}

func (s *MigrationsTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)

	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	s.path = ibctesting.NewPath(s.chainA, s.chainB)
	s.path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	s.path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	s.path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	s.path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	s.path.EndpointA.ChannelConfig.Version = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
	s.path.EndpointB.ChannelConfig.Version = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
}

func (s *MigrationsTestSuite) SetupPath() error {
	if err := s.RegisterInterchainAccount(s.path.EndpointA, ibctesting.TestAccAddress); err != nil {
		return err
	}

	if err := s.path.EndpointB.ChanOpenTry(); err != nil {
		return err
	}

	if err := s.path.EndpointA.ChanOpenAck(); err != nil {
		return err
	}

	return s.path.EndpointB.ChanOpenConfirm()
}

func (s *MigrationsTestSuite) RegisterInterchainAccount(endpoint *ibctesting.Endpoint, owner string) error {
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

func (s *MigrationsTestSuite) TestMigrateICS27ChannelCapability() {
	s.SetupTest()
	s.coordinator.SetupConnections(s.path)

	err := s.SetupPath()
	s.Require().NoError(err)

	// create additional capabilities to cover edge cases
	s.CreateMockCapabilities()

	// create and claim a new capability with ibc/mock for "channel-1"
	// note: s.SetupPath() now claims the chanel capability using icacontroller for "channel-0"
	capName := host.ChannelCapabilityPath(s.path.EndpointA.ChannelConfig.PortID, channeltypes.FormatChannelIdentifier(1))

	capability, err := s.chainA.GetSimApp().ScopedIBCKeeper.NewCapability(s.chainA.GetContext(), capName)
	s.Require().NoError(err)

	err = s.chainA.GetSimApp().ScopedICAMockKeeper.ClaimCapability(s.chainA.GetContext(), capability, capName)
	s.Require().NoError(err)

	// assert the capability is owned by the mock module
	capability, found := s.chainA.GetSimApp().ScopedICAMockKeeper.GetCapability(s.chainA.GetContext(), capName)
	s.Require().NotNil(capability)
	s.Require().True(found)

	isAuthenticated := s.chainA.GetSimApp().ScopedICAMockKeeper.AuthenticateCapability(s.chainA.GetContext(), capability, capName)
	s.Require().True(isAuthenticated)

	capability, found = s.chainA.GetSimApp().ScopedICAControllerKeeper.GetCapability(s.chainA.GetContext(), capName)
	s.Require().Nil(capability)
	s.Require().False(found)

	s.ResetMemStore() // empty the x/capability in-memory store

	err = v6.MigrateICS27ChannelCapability(
		s.chainA.GetContext(),
		s.chainA.Codec,
		s.chainA.GetSimApp().GetKey(capabilitytypes.StoreKey),
		s.chainA.GetSimApp().CapabilityKeeper,
		ibcmock.ModuleName+types.SubModuleName,
	)

	s.Require().NoError(err)

	// assert the capability is now owned by the ICS27 controller submodule
	capability, found = s.chainA.GetSimApp().ScopedICAControllerKeeper.GetCapability(s.chainA.GetContext(), capName)
	s.Require().NotNil(capability)
	s.Require().True(found)

	isAuthenticated = s.chainA.GetSimApp().ScopedICAControllerKeeper.AuthenticateCapability(s.chainA.GetContext(), capability, capName)
	s.Require().True(isAuthenticated)

	capability, found = s.chainA.GetSimApp().ScopedICAMockKeeper.GetCapability(s.chainA.GetContext(), capName)
	s.Require().Nil(capability)
	s.Require().False(found)

	// ensure channel capability for "channel-0" is still owned by the controller
	capName = host.ChannelCapabilityPath(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
	capability, found = s.chainA.GetSimApp().ScopedICAControllerKeeper.GetCapability(s.chainA.GetContext(), capName)
	s.Require().NotNil(capability)
	s.Require().True(found)

	isAuthenticated = s.chainA.GetSimApp().ScopedICAControllerKeeper.AuthenticateCapability(s.chainA.GetContext(), capability, capName)
	s.Require().True(isAuthenticated)

	s.AssertMockCapabiltiesUnchanged()
}

// CreateMockCapabilities creates an additional two capabilities used for testing purposes:
// 1. A capability with a single owner
// 2. A capability with two owners, neither of which is "ibc"
func (s *MigrationsTestSuite) CreateMockCapabilities() {
	capability, err := s.chainA.GetSimApp().ScopedIBCMockKeeper.NewCapability(s.chainA.GetContext(), "mock_one")
	s.Require().NoError(err)
	s.Require().NotNil(capability)

	capability, err = s.chainA.GetSimApp().ScopedICAMockKeeper.NewCapability(s.chainA.GetContext(), "mock_two")
	s.Require().NoError(err)
	s.Require().NotNil(capability)

	err = s.chainA.GetSimApp().ScopedIBCMockKeeper.ClaimCapability(s.chainA.GetContext(), capability, "mock_two")
	s.Require().NoError(err)
}

// AssertMockCapabiltiesUnchanged authenticates the mock capabilities created at the start of the test to ensure they remain unchanged
func (s *MigrationsTestSuite) AssertMockCapabiltiesUnchanged() {
	capability, found := s.chainA.GetSimApp().ScopedIBCMockKeeper.GetCapability(s.chainA.GetContext(), "mock_one")
	s.Require().True(found)
	s.Require().NotNil(capability)

	capability, found = s.chainA.GetSimApp().ScopedIBCMockKeeper.GetCapability(s.chainA.GetContext(), "mock_two")
	s.Require().True(found)
	s.Require().NotNil(capability)

	isAuthenticated := s.chainA.GetSimApp().ScopedICAMockKeeper.AuthenticateCapability(s.chainA.GetContext(), capability, "mock_two")
	s.Require().True(isAuthenticated)
}

// ResetMemstore removes all existing fwd and rev capability kv pairs and deletes `KeyMemInitialised` from the x/capability memstore.
// This effectively mocks a new chain binary being started. Migration code is run against persisted state only and allows the memstore to be reinitialised.
func (s *MigrationsTestSuite) ResetMemStore() {
	memStore := s.chainA.GetContext().KVStore(s.chainA.GetSimApp().GetMemKey(capabilitytypes.MemStoreKey))
	memStore.Delete(capabilitytypes.KeyMemInitialized)

	iterator := memStore.Iterator(nil, nil)
	defer sdk.LogDeferred(s.chainA.GetContext().Logger(), func() error { return iterator.Close() })

	for ; iterator.Valid(); iterator.Next() {
		memStore.Delete(iterator.Key())
	}
}
