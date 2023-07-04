package keeper_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	genesistypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/genesis/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v7/modules/core/errors"
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

	// TestVersionWithJSONEncoding defines a reusable interchainaccounts version string that uses JSON encoding for testing purposes
	TestVersionWithJSONEncoding = string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: ibctesting.FirstConnectionID,
		HostConnectionId:       ibctesting.FirstConnectionID,
		Encoding:               icatypes.EncodingProto3JSON,
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

func NewICAPath(chainA, chainB *ibctesting.TestChain, encoding string) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)

	var version string
	switch encoding {
	case icatypes.EncodingProtobuf:
		version = TestVersion
	case icatypes.EncodingProto3JSON:
		version = TestVersionWithJSONEncoding
	default:
		panic(fmt.Sprintf("unsupported encoding type: %s", encoding))
	}

	path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointA.ChannelConfig.Version = version
	path.EndpointB.ChannelConfig.Version = version

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
	suite.Run(t, new(KeeperTestSuite))
}

func (s *KeeperTestSuite) TestGetInterchainAccountAddress() {
	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf)
	s.coordinator.SetupConnections(path)

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

func (s *KeeperTestSuite) TestGetAllActiveChannels() {
	var (
		expectedChannelID = "test-channel"
		expectedPortID    = "test-port"
	)

	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf)
	s.coordinator.SetupConnections(path)

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

func (s *KeeperTestSuite) TestGetAllInterchainAccounts() {
	var (
		expectedAccAddr = "test-acc-addr"
		expectedPortID  = "test-port"
	)

	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf)
	s.coordinator.SetupConnections(path)

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

func (s *KeeperTestSuite) TestIsActiveChannel() {
	s.SetupTest()

	path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	isActive := s.chainB.GetSimApp().ICAHostKeeper.IsActiveChannel(s.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
	s.Require().True(isActive)
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

func (s *KeeperTestSuite) TestMetadataNotFound() {
	var (
		invalidPortID    = "invalid-port"
		invalidChannelID = "invalid-channel"
	)

	_, err := s.chainB.GetSimApp().ICAHostKeeper.GetAppMetadata(s.chainB.GetContext(), invalidPortID, invalidChannelID)
	s.Require().ErrorIs(err, ibcerrors.ErrNotFound)
	s.Require().Contains(err.Error(), fmt.Sprintf("app version not found for port %s and channel %s", invalidPortID, invalidChannelID))
}

func (s *KeeperTestSuite) TestParams() {
	expParams := types.DefaultParams()

	params := s.chainA.GetSimApp().ICAHostKeeper.GetParams(s.chainA.GetContext())
	s.Require().Equal(expParams, params)

	testCases := []struct {
		name    string
		input   types.Params
		expPass bool
	}{
		{"success: set default params", types.DefaultParams(), true},
		{"success: non-default params", types.NewParams(!types.DefaultHostEnabled, []string{"/cosmos.staking.v1beta1.MsgDelegate"}), true},
		{"success: set empty byte for allow messages", types.NewParams(true, nil), true},
		{"failure: set empty string for allow messages", types.NewParams(true, []string{""}), false},
		{"failure: set space string for allow messages", types.NewParams(true, []string{" "}), false},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()
			err := tc.input.Validate()
			s.chainA.GetSimApp().ICAHostKeeper.SetParams(ctx, tc.input)
			if tc.expPass {
				s.Require().NoError(err)
				expected := tc.input
				p := s.chainA.GetSimApp().ICAHostKeeper.GetParams(ctx)
				s.Require().Equal(expected, p)
			} else {
				s.Require().Error(err)
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
		s.chainA.GetSimApp().ICAHostKeeper.GetParams(ctx)
	})
}
