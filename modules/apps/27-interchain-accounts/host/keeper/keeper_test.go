package keeper_test

import (
	"fmt"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/runtime"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	genesistypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/genesis/types"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
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

func NewICAPath(chainA, chainB *ibctesting.TestChain, encoding string, ordering channeltypes.Order) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)

	var version string
	switch encoding {
	case icatypes.EncodingProtobuf:
		version = TestVersion
	case icatypes.EncodingProto3JSON:
		version = TestVersionWithJSONEncoding
	default:
		panic(fmt.Errorf("unsupported encoding type: %s", encoding))
	}

	path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointA.ChannelConfig.Order = ordering
	path.EndpointB.ChannelConfig.Order = ordering
	path.EndpointA.ChannelConfig.Version = version
	path.EndpointB.ChannelConfig.Version = version

	return path
}

// SetupICAPath invokes the InterchainAccounts entrypoint and subsequent channel handshake handlers
func SetupICAPath(path *ibctesting.Path, owner string) error {
	path.EndpointA.IncrementNextChannelSequence()

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

	if err := endpoint.Chain.GetSimApp().ICAControllerKeeper.RegisterInterchainAccount(endpoint.Chain.GetContext(), endpoint.ConnectionID, owner, endpoint.ChannelConfig.Version, endpoint.ChannelConfig.Order); err != nil {
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
		panicMsg      string
	}{
		{"success", func() {
			keeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(types.StoreKey)),
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				s.chainA.GetSimApp().AccountKeeper,
				s.chainA.GetSimApp().MsgServiceRouter(),
				s.chainA.GetSimApp().GRPCQueryRouter(),
				s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(),
			)
		}, ""},
		{"failure: interchain accounts module account does not exist", func() {
			keeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(types.StoreKey)),
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				authkeeper.AccountKeeper{}, // empty account keeper
				s.chainA.GetSimApp().MsgServiceRouter(),
				s.chainA.GetSimApp().GRPCQueryRouter(),
				s.chainA.GetSimApp().ICAHostKeeper.GetAuthority(),
			)
		}, "the Interchain Accounts module account has not been set"},
		{"failure: empty mock staking keeper", func() {
			keeper.NewKeeper(
				s.chainA.GetSimApp().AppCodec(),
				runtime.NewKVStoreService(s.chainA.GetSimApp().GetKey(types.StoreKey)),
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper,
				s.chainA.GetSimApp().AccountKeeper,
				s.chainA.GetSimApp().MsgServiceRouter(),
				s.chainA.GetSimApp().GRPCQueryRouter(),
				"", // authority
			)
		}, "authority must be non-empty"},
	}

	for _, tc := range testCases {
		s.SetupTest()

		s.Run(tc.name, func() {
			if tc.panicMsg == "" {
				s.Require().NotPanics(
					tc.instantiateFn,
				)
			} else {
				s.Require().PanicsWithError(
					tc.panicMsg,
					tc.instantiateFn,
				)
			}
		})
	}
}

func (s *KeeperTestSuite) TestNewModuleQuerySafeAllowList() {
	// Currently, all queries in bank, staking, auth, and circuit are marked safe
	// Notably, the gov and distribution modules are not marked safe

	var allowList []string
	s.Require().NotPanics(func() {
		allowList = keeper.NewModuleQuerySafeAllowList()
	})

	s.Require().NotEmpty(allowList)
	s.Require().Contains(allowList, "/cosmos.bank.v1beta1.Query/Balance")
	s.Require().Contains(allowList, "/cosmos.bank.v1beta1.Query/AllBalances")
	s.Require().Contains(allowList, "/cosmos.staking.v1beta1.Query/Validator")
	s.Require().Contains(allowList, "/cosmos.staking.v1beta1.Query/Validators")
	s.Require().Contains(allowList, "/cosmos.auth.v1beta1.Query/Accounts")
	s.Require().Contains(allowList, "/cosmos.auth.v1beta1.Query/ModuleAccountByName")
	s.Require().Contains(allowList, "/ibc.core.client.v1.Query/VerifyMembership")
	s.Require().NotContains(allowList, "/cosmos.gov.v1beta1.Query/Proposals")
	s.Require().NotContains(allowList, "/cosmos.gov.v1.Query/Proposals")
	s.Require().NotContains(allowList, "/cosmos.distribution.v1beta1.Query/Params")
	s.Require().NotContains(allowList, "/cosmos.distribution.v1beta1.Query/DelegationRewards")
}

func (s *KeeperTestSuite) TestGetInterchainAccountAddress() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest()

		path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf, ordering)
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

		path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf, ordering)
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

		path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf, ordering)
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

		path := NewICAPath(s.chainA, s.chainB, icatypes.EncodingProtobuf, ordering)
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
		name   string
		input  types.Params
		errMsg string
	}{
		{"success: set default params", types.DefaultParams(), ""},
		{"success: non-default params", types.NewParams(!types.DefaultHostEnabled, []string{"/cosmos.staking.v1beta1.MsgDelegate"}), ""},
		{"success: set empty byte for allow messages", types.NewParams(true, nil), ""},
		{"failure: set empty string for allow messages", types.NewParams(true, []string{""}), "parameter must not contain empty strings"},
		{"failure: set space string for allow messages", types.NewParams(true, []string{" "}), "parameter must not contain empty strings"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			ctx := s.chainA.GetContext()
			err := tc.input.Validate()
			s.chainA.GetSimApp().ICAHostKeeper.SetParams(ctx, tc.input)
			if tc.errMsg == "" {
				s.Require().NoError(err)
				expected := tc.input
				p := s.chainA.GetSimApp().ICAHostKeeper.GetParams(ctx)
				s.Require().Equal(expected, p)
			} else {
				s.Require().ErrorContains(err, tc.errMsg)
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

func (s *KeeperTestSuite) TestWithICS4Wrapper() {
	s.SetupTest()

	// test if the ics4 wrapper is the channel keeper initially
	ics4Wrapper := s.chainA.GetSimApp().ICAHostKeeper.GetICS4Wrapper()

	_, isChannelKeeper := ics4Wrapper.(*channelkeeper.Keeper)
	s.Require().True(isChannelKeeper)
	s.Require().IsType((*channelkeeper.Keeper)(nil), ics4Wrapper)

	// set the ics4 wrapper to the channel keeper
	s.chainA.GetSimApp().ICAHostKeeper.WithICS4Wrapper(nil)
	ics4Wrapper = s.chainA.GetSimApp().ICAHostKeeper.GetICS4Wrapper()
	s.Require().Nil(ics4Wrapper)
}
