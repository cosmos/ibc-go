package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	genesistypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	keeper "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/keeper"
	ratelimittypes "github.com/cosmos/ibc-go/v10/modules/apps/rate-limiting/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

// TestOwnerAddress defines a reusable bech32 address for testing purposes
var (
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"
	TestPortID, _    = icatypes.NewControllerPortID(TestOwnerAddress)

	// TestVersion defines a reusable interchainaccounts version string for testing purposes
	TestVersion = string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: ibctesting.FirstConnectionID,
		HostConnectionId:       ibctesting.FirstConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}))
)

// MockQueryRouter is a mock implementation of the QueryRouter interface
type MockQueryRouter struct{}

func (MockQueryRouter) Route(path string) func(ctx sdk.Context, req interface{}) ([]byte, error) {
	return func(ctx sdk.Context, req any) ([]byte, error) {
		return nil, nil
	}
}

// MockMsgRouter is a mock implementation of the MessageRouter interface
type MockMsgRouter struct{}

func (MockMsgRouter) Handler(msg sdk.Msg) func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		return nil, nil
	}
}

func NewICAPath(chainA, chainB *ibctesting.TestChain, ordering channeltypes.Order) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointA.ChannelConfig.Order = ordering
	path.EndpointB.ChannelConfig.Order = ordering
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

	if err := endpoint.Chain.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, owner, TestVersion, endpoint.ChannelConfig.Order); err != nil {
		return err
	}

	// commit state changes for proof verification
	endpoint.Chain.NextBlock()

	// update port/channel ids
	endpoint.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
	endpoint.ChannelConfig.PortID = portID

	return nil
}

type KeeperTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

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

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestNewKeeper() {
	testCases := []struct {
		name          string
		instantiateFn func()
		panicMsg      string
	}{
		{
			name: "success",
			instantiateFn: func() {
				keeper.NewKeeper(
					s.chainA.GetSimApp().AppCodec(),
					runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ratelimittypes.StoreKey)),
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper, // This is now used as ics4Wrapper
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					s.chainA.GetSimApp().IBCKeeper.ClientKeeper, // Add clientKeeper
					s.chainA.GetSimApp().BankKeeper,
					s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(),
				)
			},
			panicMsg: "",
		},
		{
			name: "failure: empty authority",
			instantiateFn: func() {
				keeper.NewKeeper(
					s.chainA.GetSimApp().AppCodec(),
					runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(ratelimittypes.StoreKey)),
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper, // ics4Wrapper
					s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
					s.chainA.GetSimApp().IBCKeeper.ClientKeeper, // clientKeeper
					s.chainA.GetSimApp().BankKeeper,
					"", // empty authority
				)
			},
			panicMsg: "authority must be non-empty",
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.SetupTest()

		s.Run(tc.name, func() {
			if tc.panicMsg == "" {
				s.Require().NotPanics(tc.instantiateFn)
			} else {
				s.Require().PanicsWithError(tc.panicMsg, tc.instantiateFn)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetInterchainAccountAddress() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		counterpartyPortID := path.EndpointA.ChannelConfig.PortID

		retrievedAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, counterpartyPortID)
		s.Require().True(found)
		s.Require().NotEmpty(retrievedAddr)

		retrievedAddr, found = s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, "invalid port")
		s.Require().False(found)
		s.Require().Empty(retrievedAddr)
	}
}

func (s *KeeperTestSuite) TestGetAllActiveChannels() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		var (
			expectedChannelID = "test-channel"
			expectedPortID    = "test-port"
		)

		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		s.chainB.GetSimApp().ICAHostKeeper.SetActiveChannelID(s.chainB.GetContext(), ibctesting.FirstConnectionID, expectedPortID, expectedChannelID)

		expectedChannels := []genesistypes.ActiveChannel{
			{
				ConnectionId: ibctesting.FirstConnectionID,
				PortId:       path.EndpointA.ChannelConfig.PortID,
				ChannelId:    path.EndpointB.ChannelID,
			},
			{
				ConnectionId: ibctesting.FirstConnectionID,
				PortId:       expectedPortID,
				ChannelId:    expectedChannelID,
			},
		}

		activeChannels := s.chainB.GetSimApp().ICAHostKeeper.GetAllActiveChannels(s.chainB.GetContext())
		s.Require().Len(activeChannels, len(expectedChannels))
		s.Require().Equal(expectedChannels, activeChannels)
	}
}

func (s *KeeperTestSuite) TestGetAllInterchainAccounts() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		var (
			expectedAccAddr = "test-acc-addr"
			expectedPortID  = "test-port"
		)

		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		interchainAccAddr, exists := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), path.EndpointB.ConnectionID, path.EndpointA.ChannelConfig.PortID)
		s.Require().True(exists)

		s.chainB.GetSimApp().ICAHostKeeper.SetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, expectedPortID, expectedAccAddr)

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

		interchainAccounts := s.chainB.GetSimApp().ICAHostKeeper.GetAllInterchainAccounts(s.chainB.GetContext())
		s.Require().Len(interchainAccounts, len(expectedAccounts))
		s.Require().Equal(expectedAccounts, interchainAccounts)
	}
}

func (s *KeeperTestSuite) TestIsActiveChannel() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		isActive := s.chainB.GetSimApp().ICAHostKeeper.IsActiveChannel(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
		s.Require().True(isActive)
	}
}

func (s *KeeperTestSuite) TestSetInterchainAccountAddress() {
	var (
		expectedAccAddr = "test-acc-addr"
		expectedPortID  = "test-port"
	)

	s.chainB.GetSimApp().ICAHostKeeper.SetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, expectedPortID, expectedAccAddr)

	retrievedAddr, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), ibctesting.FirstConnectionID, expectedPortID)
	s.Require().True(found)
	s.Require().Equal(expectedAccAddr, retrievedAddr)
}
