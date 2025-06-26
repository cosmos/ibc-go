package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/runtime"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	genesistypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
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
	testifysuite.Suite

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

func TestKeeperTestSuite(t *testing.T) {
	testifysuite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestNewKeeper() {
	testCases := []struct {
		name          string
		instantiateFn func()
		errMsg        string
	}{
		{"success", func() {
			keeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(types.StoreKey)),
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				s.chainA.GetSimApp().MsgServiceRouter(),
				s.chainA.GetSimApp().ICAControllerKeeper.GetAuthority(),
			)
		}, ""},
		{"failure: empty authority", func() {
			keeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(types.StoreKey)),
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				s.chainA.GetSimApp().MsgServiceRouter(),
				"", // authority
			)
		}, "authority must be non-empty"},
	}

	for _, tc := range testCases {
		s.SetupTest()

		s.Run(tc.name, func() {
			if tc.errMsg == "" {
				s.Require().NotPanics(
					tc.instantiateFn,
				)
			} else {
				s.Require().PanicsWithError(
					tc.errMsg,
					tc.instantiateFn,
				)
			}
		})
	}
}

func (s *KeeperTestSuite) TestGetAllPorts() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		expectedPorts := []string{TestPortID}

		ports := s.chainA.GetSimApp().ICAControllerKeeper.GetAllPorts(s.chainA.GetContext())
		s.Require().Len(ports, len(expectedPorts))
		s.Require().Equal(expectedPorts, ports)
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

		retrievedAddr, found := s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), ibctesting.FirstConnectionID, counterpartyPortID)
		s.Require().True(found)
		s.Require().NotEmpty(retrievedAddr)

		retrievedAddr, found = s.chainA.GetSimApp().ICAControllerKeeper.GetInterchainAccountAddress(s.chainA.GetContext(), "invalid conn", "invalid port")
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
}

func (s *KeeperTestSuite) TestIsActiveChannel() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, ordering)
		owner := TestOwnerAddress
		path.SetupConnections()

		err := SetupICAPath(path, owner)
		s.Require().NoError(err)
		portID := path.EndpointA.ChannelConfig.PortID

		isActive := s.chainA.GetSimApp().ICAControllerKeeper.IsActiveChannel(s.chainA.GetContext(), ibctesting.FirstConnectionID, portID)
		s.Require().True(isActive)
	}
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
		name  string
		input types.Params
	}{
		{"success: set params false", types.NewParams(false)},
		{"success: set params true", types.NewParams(true)},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()

			s.chainA.GetSimApp().ICAControllerKeeper.SetParams(ctx, tc.input)
			expected := tc.input
			p := s.chainA.GetSimApp().ICAControllerKeeper.GetParams(ctx)
			s.Require().Equal(expected, p)
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

func (s *KeeperTestSuite) TestWithICS4Wrapper() {
	s.SetupTest()

	// test if the ics4 wrapper is the channel keeper initially
	ics4Wrapper := s.chainA.GetSimApp().ICAControllerKeeper.GetICS4Wrapper()

	_, isChannelKeeper := ics4Wrapper.(*channelkeeper.Keeper)
	s.Require().True(isChannelKeeper)
	s.Require().IsType((*channelkeeper.Keeper)(nil), ics4Wrapper)

	// set the ics4 wrapper to the channel keeper
	s.chainA.GetSimApp().ICAControllerKeeper.WithICS4Wrapper(nil)
	ics4Wrapper = s.chainA.GetSimApp().ICAControllerKeeper.GetICS4Wrapper()
	s.Require().Nil(ics4Wrapper)
}
