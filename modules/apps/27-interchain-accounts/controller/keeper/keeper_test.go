package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// TestPortID defines a reusable port identifier for testing purposes
	TestPortID, _ = icatypes.NewControllerPortID(TestOwnerAddress)

	// TestVersion defines a reusable interchainaccounts version string for testing purposes
	TestVersion = string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: ibctesting.FirstConnectionID,
		HostConnectionId:       ibctesting.FirstConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}))
)

type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func (s *KeeperTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))
}

func NewICAPath(chainA, chainB *ibctesting.TestChain) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointA.ChannelConfig.Version = TestVersion
	path.EndpointB.ChannelConfig.Version = TestVersion

	return path
}

// SetupICAPath invokes the InterchainAccounts entrypoint and subsequent channel handshake handlers
func SetupICAPath(path *ibctesting.Path, owner string) error {
	if err := RegisterInterchainAccount(path.EndpointA, owner); err != nil {
		return err
	}

	if err := path.EndpointB.ChanOpenTry(); err != nil {
		return err
	}

	if err := path.EndpointA.ChanOpenAck(); err != nil {
		return err
	}

	return path.EndpointB.ChanOpenConfirm()
}

// RegisterInterchainAccount is a helper function for starting the channel handshake
func RegisterInterchainAccount(endpoint *ibctesting.Endpoint, owner string) error {
	portID, err := icatypes.NewControllerPortID(owner)
	if err != nil {
		return err
	}

	channelSequence := endpoint.Chain.App.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(endpoint.Chain.GetContext())

	if err := endpoint.Chain.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, owner, TestVersion); err != nil {
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
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestGetAllPorts() {
	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	expectedPorts := []string{TestPortID}

	ports := s.chainA.GetSimApp().ICAControllerKeeper.GetAllPorts(s.chainA.GetContext())
	s.Require().Len(ports, len(expectedPorts))
	s.Require().Equal(expectedPorts, ports)
}

func (s *KeeperTestSuite) TestGetInterchainAccountAddress() {
	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	counterpartyPortID := path.EndpointA.ChannelConfig.PortID

	retrievedAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, counterpartyPortID)
	s.Require().True(found)
	s.Require().NotEmpty(retrievedAddr)

	retrievedAddr, found = s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), "invalid conn", "invalid port")
	s.Require().False(found)
	s.Require().Empty(retrievedAddr)
}

func (s *KeeperTestSuite) TestGetAllActiveChannels() {
	var (
		expectedChannelID = "test-channel"
		expectedPortID    = "test-port"
	)

	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	s.chainA.GetSimApp().ICAControllerKeeper.SetActiveChannelID(s.chainA.GetContext(), ibctesting.FirstConnectionID, expectedPortID, expectedChannelID)

	expectedChannels := []genesistypes.ActiveChannel{
		{
			ConnectionId:        ibctesting.FirstConnectionID,
			PortId:              TestPortID,
			ChannelId:           path.EndpointA.ChannelID,
			IsMiddlewareEnabled: true,
		},
		{
			ConnectionId:        ibctesting.FirstConnectionID,
			PortId:              expectedPortID,
			ChannelId:           expectedChannelID,
			IsMiddlewareEnabled: false,
		},
	}

	activeChannels := s.chainA.GetSimApp().ICAControllerKeeper.GetAllActiveChannels(s.chainA.GetContext())
	s.Require().Len(activeChannels, len(expectedChannels))
	s.Require().Equal(expectedChannels, activeChannels)
}

func (s *KeeperTestSuite) TestGetAllInterchainAccounts() {
	var (
		expectedAccAddr = "test-acc-addr"
		expectedPortID  = "test-port"
	)

	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	interchainAccAddr, exists := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
	s.Require().True(exists)

	s.chainA.GetSimApp().ICAControllerKeeper.SetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, expectedPortID, expectedAccAddr)

	expectedAccounts := []genesistypes.RegisteredInterchainAccount{
		{
			ConnectionId:   ibctesting.FirstConnectionID,
			PortId:         TestPortID,
			AccountAddress: interchainAccAddr,
		},
		{
			ConnectionId:   ibctesting.FirstConnectionID,
			PortId:         expectedPortID,
			AccountAddress: expectedAccAddr,
		},
	}

	interchainAccounts := s.chainA.GetSimApp().ICAControllerKeeper.GetAllInterchainAccounts(s.chainA.GetContext())
	s.Require().Len(interchainAccounts, len(expectedAccounts))
	s.Require().Equal(expectedAccounts, interchainAccounts)
}

func (s *KeeperTestSuite) TestIsActiveChannel() {
	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB)
	owner := TestOwnerAddress
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, owner)
	s.Require().NoError(err)
	portID := path.EndpointA.ChannelConfig.PortID

	isActive := s.chainA.GetSimApp().ICAControllerKeeper.IsActiveChannel(s.chainA.GetContext(), ibctesting.FirstConnectionID, portID)
	s.Require().Equal(isActive, true)
}

func (s *KeeperTestSuite) TestSetInterchainAccountAddress() {
	var (
		expectedAccAddr = "test-acc-addr"
		expectedPortID  = "test-port"
	)

	s.chainA.GetSimApp().ICAControllerKeeper.SetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, expectedPortID, expectedAccAddr)

	retrievedAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, expectedPortID)
	s.Require().True(found)
	s.Require().Equal(expectedAccAddr, retrievedAddr)
}

func (s *KeeperTestSuite) TestSetAndGetParams() {
	testCases := []struct {
		name    string
		input   types.Params
		expPass bool
	}{
		// it is not possible to set invalid booleans
		{"success: set params false", types.NewParams(false), true},
		{"success: set params true", types.NewParams(true), true},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()
			if tc.expPass {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(ctx, tc.input)
				expected := tc.input
				p := s.chainA.GetSimApp().ICAControllerKeeper.GetParams(ctx)
				s.Require().Equal(expected, p)
			} else { // currently not possible to set invalid params
				s.Require().Panics(func() {
					s.chainA.GetSimApp().ICAControllerKeeper.SetParams(ctx, tc.input)
				})
			}
		})
	}
}

func (s *KeeperTestSuite) TestUnsetParams() {
	s.SetupTest()

	ctx := s.chainA.GetContext()
	store := s.chainA.GetContext().KVStore(s.chainA.GetSimApp().GetKey(types.SubModuleName))
	store.Delete([]byte(types.ParamsKey))

	s.Require().Panics(func() {
		s.chainA.GetSimApp().ICAControllerKeeper.GetParams(ctx)
	})
}

func (s *KeeperTestSuite) TestGetAuthority() {
	s.SetupTest()

	authority := s.chainA.GetSimApp().ICAControllerKeeper.GetAuthority()
	expectedAuth := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	s.Require().Equal(expectedAuth, authority)
}
