package controller_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller"
	controllerkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	fee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
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

type InterchainAccountsTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func TestICATestSuite(t *testing.T) {
	suite.Run(t, new(InterchainAccountsTestSuite))
}

func (s *InterchainAccountsTestSuite) SetupTest() {
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
	endpoint.ChannelConfig.Version = TestVersion

	return nil
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

func (s *InterchainAccountsTestSuite) TestOnChanOpenInit() {
	var (
		channel  *channeltypes.Channel
		isNilApp bool
		path     *ibctesting.Path
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"ICA auth module does not claim channel capability", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string, chanCap *capabilitytypes.Capability,
					counterparty channeltypes.Counterparty, version string,
				) (string, error) {
					if chanCap != nil {
						return "", fmt.Errorf("channel capability should be nil")
					}

					return version, nil
				}
			}, true,
		},
		{
			"ICA auth module modification of channel version is ignored", func() {
				// NOTE: explicitly modify the channel version via the auth module callback,
				// ensuring the expected JSON encoded metadata is not modified upon return
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string, chanCap *capabilitytypes.Capability,
					counterparty channeltypes.Counterparty, version string,
				) (string, error) {
					return "invalid-version", nil
				}
			}, true,
		},
		{
			"controller submodule disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			}, false,
		},
		{
			"ICA OnChanOpenInit fails - UNORDERED channel", func() {
				channel.Ordering = channeltypes.UNORDERED
			}, false,
		},
		{
			"ICA auth module callback fails", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string, chanCap *capabilitytypes.Capability,
					counterparty channeltypes.Counterparty, version string,
				) (string, error) {
					return "", fmt.Errorf("mock ica auth fails")
				}
			}, false,
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, true,
		},
		{
			"middleware disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string, chanCap *capabilitytypes.Capability,
					counterparty channeltypes.Counterparty, version string,
				) (string, error) {
					return "", fmt.Errorf("error should be unreachable")
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			isNilApp = false

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			// mock init interchain account
			portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
			s.Require().NoError(err)

			portCap := s.chainA.GetSimApp().IBCKeeper.PortKeeper.BindPort(s.chainA.GetContext(), portID)
			s.chainA.GetSimApp().ICAControllerKeeper.ClaimCapability(s.chainA.GetContext(), portCap, host.PortPath(portID)) //nolint:errcheck // checking this error isn't needed for the test

			path.EndpointA.ChannelConfig.PortID = portID
			path.EndpointA.ChannelID = ibctesting.FirstChannelID

			s.chainA.GetSimApp().ICAControllerKeeper.SetMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

			// default values
			counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
			channel = &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.ORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{path.EndpointA.ConnectionID},
				Version:        path.EndpointA.ChannelConfig.Version,
			}

			tc.malleate() // malleate mutates test data

			// ensure channel on chainA is set in state
			s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, *channel)

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
			s.Require().NoError(err)

			chanCap, err := s.chainA.App.GetScopedIBCKeeper().NewCapability(s.chainA.GetContext(), host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			if isNilApp {
				cbs = controller.NewIBCMiddleware(nil, s.chainA.GetSimApp().ICAControllerKeeper)
			}

			version, err := cbs.OnChanOpenInit(s.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, chanCap, channel.Counterparty, channel.GetVersion(),
			)

			if tc.expPass {
				s.Require().Equal(icatypes.NewDefaultMetadataString(path.EndpointA.ConnectionID, path.EndpointB.ConnectionID), version)
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// Test initiating a ChanOpenTry using the controller chain instead of the host chain
// ChainA is the controller chain. ChainB creates a controller port as well,
// attempting to trick chainA.
// Sending a MsgChanOpenTry will never reach the application callback due to
// core IBC checks not passing, so a call to the application callback is also
// done directly.
func (s *InterchainAccountsTestSuite) TestChanOpenTry() {
	s.SetupTest() // reset
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
	s.Require().NoError(err)

	// chainB also creates a controller port
	err = RegisterInterchainAccount(path.EndpointB, TestOwnerAddress)
	s.Require().NoError(err)

	err = path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
	proofInit, proofHeight := path.EndpointB.Chain.QueryProof(channelKey)

	// use chainA (controller) for ChanOpenTry
	msg := channeltypes.NewMsgChannelOpenTry(path.EndpointA.ChannelConfig.PortID, TestVersion, channeltypes.ORDERED, []string{path.EndpointA.ConnectionID}, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, TestVersion, proofInit, proofHeight, icatypes.ModuleName)
	handler := s.chainA.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err = handler(s.chainA.GetContext(), msg)

	s.Require().Error(err)

	// call application callback directly
	module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointB.ChannelConfig.PortID)
	s.Require().NoError(err)

	cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	s.Require().True(ok)

	counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
	chanCap, found := s.chainA.App.GetScopedIBCKeeper().GetCapability(s.chainA.GetContext(), host.ChannelCapabilityPath(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID))
	s.Require().True(found)

	version, err := cbs.OnChanOpenTry(
		s.chainA.GetContext(), path.EndpointA.ChannelConfig.Order, []string{path.EndpointA.ConnectionID},
		path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, chanCap,
		counterparty, path.EndpointB.ChannelConfig.Version,
	)
	s.Require().Error(err)
	s.Require().Equal("", version)
}

func (s *InterchainAccountsTestSuite) TestOnChanOpenAck() {
	var (
		path     *ibctesting.Path
		isNilApp bool
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"controller submodule disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			}, false,
		},
		{
			"ICA OnChanOpenACK fails - invalid version", func() {
				path.EndpointB.ChannelConfig.Version = "invalid|version"
			}, false,
		},
		{
			"ICA auth module callback fails", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenAck = func(
					ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string,
				) error {
					return fmt.Errorf("mock ica auth fails")
				}
			}, false,
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, true,
		},
		{
			"middleware disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenAck = func(
					ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string,
				) error {
					return fmt.Errorf("error should be unreachable")
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset
			isNilApp = false

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
			s.Require().NoError(err)

			err = path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			err = cbs.OnChanOpenAck(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelID, path.EndpointB.ChannelConfig.Version)

			if isNilApp {
				cbs = controller.NewIBCMiddleware(nil, s.chainA.GetSimApp().ICAControllerKeeper)
			}

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// Test initiating a ChanOpenConfirm using the controller chain instead of the host chain
// ChainA is the controller chain. ChainB is the host chain
// Sending a MsgChanOpenConfirm will never reach the application callback due to
// core IBC checks not passing, so a call to the application callback is also
// done directly.
func (s *InterchainAccountsTestSuite) TestChanOpenConfirm() {
	s.SetupTest() // reset
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
	s.Require().NoError(err)

	err = path.EndpointB.ChanOpenTry()
	s.Require().NoError(err)

	// chainB maliciously sets channel to OPEN
	channel := channeltypes.NewChannel(channeltypes.OPEN, channeltypes.ORDERED, channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, TestVersion)
	s.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel)

	// commit state changes so proof can be created
	s.chainB.NextBlock()

	err = path.EndpointA.UpdateClient()
	s.Require().NoError(err)

	// query proof from ChainB
	channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
	proofAck, proofHeight := path.EndpointB.Chain.QueryProof(channelKey)

	// use chainA (controller) for ChanOpenConfirm
	msg := channeltypes.NewMsgChannelOpenConfirm(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, proofAck, proofHeight, icatypes.ModuleName)
	handler := s.chainA.GetSimApp().MsgServiceRouter().Handler(msg)
	_, err = handler(s.chainA.GetContext(), msg)

	s.Require().Error(err)

	// call application callback directly
	module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
	s.Require().NoError(err)

	cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	s.Require().True(ok)

	err = cbs.OnChanOpenConfirm(
		s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
	)
	s.Require().Error(err)
}

// OnChanCloseInit on controller (chainA)
func (s *InterchainAccountsTestSuite) TestOnChanCloseInit() {
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
	s.Require().NoError(err)

	cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	s.Require().True(ok)

	err = cbs.OnChanCloseInit(
		s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
	)

	s.Require().Error(err)
}

func (s *InterchainAccountsTestSuite) TestOnChanCloseConfirm() {
	var (
		path     *ibctesting.Path
		isNilApp bool
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success", func() {}, true,
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			isNilApp = false

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data
			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			if isNilApp {
				cbs = controller.NewIBCMiddleware(nil, s.chainA.GetSimApp().ICAControllerKeeper)
			}

			err = cbs.OnChanCloseConfirm(
				s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *InterchainAccountsTestSuite) TestOnRecvPacket() {
	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"ICA OnRecvPacket fails with ErrInvalidChannelFlow", func() {}, false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest() // reset

			path := NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			packet := channeltypes.NewPacket(
				[]byte("empty packet data"),
				s.chainB.SenderAccount.GetSequence(),
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			ack := cbs.OnRecvPacket(s.chainA.GetContext(), packet, nil)
			s.Require().Equal(tc.expPass, ack.Success())
		})
	}
}

func (s *InterchainAccountsTestSuite) TestOnAcknowledgementPacket() {
	var (
		path     *ibctesting.Path
		isNilApp bool
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"controller submodule disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			}, false,
		},
		{
			"ICA auth module callback fails", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnAcknowledgementPacket = func(
					ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress,
				) error {
					return fmt.Errorf("mock ica auth fails")
				}
			}, false,
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, true,
		},
		{
			"middleware disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnAcknowledgementPacket = func(
					ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress,
				) error {
					return fmt.Errorf("error should be unreachable")
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest() // reset
			isNilApp = false

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			packet := channeltypes.NewPacket(
				[]byte("empty packet data"),
				s.chainA.SenderAccount.GetSequence(),
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			tc.malleate() // malleate mutates test data

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			if isNilApp {
				cbs = controller.NewIBCMiddleware(nil, s.chainA.GetSimApp().ICAControllerKeeper)
			}

			err = cbs.OnAcknowledgementPacket(s.chainA.GetContext(), packet, []byte("ack"), nil)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *InterchainAccountsTestSuite) TestOnTimeoutPacket() {
	var (
		path     *ibctesting.Path
		isNilApp bool
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"controller submodule disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			}, false,
		},
		{
			"ICA auth module callback fails", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnTimeoutPacket = func(
					ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress,
				) error {
					return fmt.Errorf("mock ica auth fails")
				}
			}, false,
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, true,
		},
		{
			"middleware disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnTimeoutPacket = func(
					ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress,
				) error {
					return fmt.Errorf("error should be unreachable")
				}
			}, true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest() // reset
			isNilApp = false

			path = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			s.Require().NoError(err)

			packet := channeltypes.NewPacket(
				[]byte("empty packet data"),
				s.chainA.SenderAccount.GetSequence(),
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			tc.malleate() // malleate mutates test data

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			if isNilApp {
				cbs = controller.NewIBCMiddleware(nil, s.chainA.GetSimApp().ICAControllerKeeper)
			}

			err = cbs.OnTimeoutPacket(s.chainA.GetContext(), packet, nil)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *InterchainAccountsTestSuite) TestSingleHostMultipleControllers() {
	var (
		pathAToB *ibctesting.Path
		pathCToB *ibctesting.Path
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.msg, func() {
			s.SetupTest() // reset

			// Setup a new path from A(controller) -> B(host)
			pathAToB = NewICAPath(s.chainA, s.chainB)
			s.coordinator.SetupConnections(pathAToB)

			err := SetupICAPath(pathAToB, TestOwnerAddress)
			s.Require().NoError(err)

			// Setup a new path from C(controller) -> B(host)
			pathCToB = NewICAPath(s.chainC, s.chainB)
			s.coordinator.SetupConnections(pathCToB)

			// NOTE: Here the version metadata is overridden to include to the next host connection sequence (i.e. chainB's connection to chainC)
			// SetupICAPath() will set endpoint.ChannelConfig.Version to TestVersion
			TestVersion = string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
				Version:                icatypes.Version,
				ControllerConnectionId: pathCToB.EndpointA.ConnectionID,
				HostConnectionId:       pathCToB.EndpointB.ConnectionID,
				Encoding:               icatypes.EncodingProtobuf,
				TxType:                 icatypes.TxTypeSDKMultiMsg,
			}))

			err = SetupICAPath(pathCToB, TestOwnerAddress)
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			accAddressChainA, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), pathAToB.EndpointB.ConnectionID, pathAToB.EndpointA.ChannelConfig.PortID)
			s.Require().True(found)

			accAddressChainC, found := s.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(s.chainB.GetContext(), pathCToB.EndpointB.ConnectionID, pathCToB.EndpointA.ChannelConfig.PortID)
			s.Require().True(found)

			s.Require().NotEqual(accAddressChainA, accAddressChainC)

			chainAChannelID, found := s.chainB.GetSimApp().ICAHostKeeper.GetActiveChannelID(s.chainB.GetContext(), pathAToB.EndpointB.ConnectionID, pathAToB.EndpointA.ChannelConfig.PortID)
			s.Require().True(found)

			chainCChannelID, found := s.chainB.GetSimApp().ICAHostKeeper.GetActiveChannelID(s.chainB.GetContext(), pathCToB.EndpointB.ConnectionID, pathCToB.EndpointA.ChannelConfig.PortID)
			s.Require().True(found)

			s.Require().NotEqual(chainAChannelID, chainCChannelID)
		})
	}
}

func (s *InterchainAccountsTestSuite) TestGetAppVersion() {
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, TestOwnerAddress)
	s.Require().NoError(err)

	module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID)
	s.Require().NoError(err)

	cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
	s.Require().True(ok)

	controllerStack := cbs.(fee.IBCMiddleware)
	appVersion, found := controllerStack.GetAppVersion(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	s.Require().True(found)
	s.Require().Equal(path.EndpointA.ChannelConfig.Version, appVersion)
}

func (s *InterchainAccountsTestSuite) TestInFlightHandshakeRespectsGoAPICaller() {
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	// initiate a channel handshake such that channel.State == INIT
	err := RegisterInterchainAccount(path.EndpointA, s.chainA.SenderAccount.GetAddress().String())
	s.Require().NoError(err)

	// attempt to start a second handshake via the controller msg server
	msgServer := controllerkeeper.NewMsgServerImpl(&s.chainA.GetSimApp().ICAControllerKeeper)
	msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, s.chainA.SenderAccount.GetAddress().String(), TestVersion)

	res, err := msgServer.RegisterInterchainAccount(s.chainA.GetContext(), msgRegisterInterchainAccount)
	s.Require().Error(err)
	s.Require().Nil(res)
}

func (s *InterchainAccountsTestSuite) TestInFlightHandshakeRespectsMsgServerCaller() {
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	// initiate a channel handshake such that channel.State == INIT
	msgServer := controllerkeeper.NewMsgServerImpl(&s.chainA.GetSimApp().ICAControllerKeeper)
	msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, s.chainA.SenderAccount.GetAddress().String(), TestVersion)

	res, err := msgServer.RegisterInterchainAccount(s.chainA.GetContext(), msgRegisterInterchainAccount)
	s.Require().NotNil(res)
	s.Require().NoError(err)

	// attempt to start a second handshake via the legacy Go API
	err = RegisterInterchainAccount(path.EndpointA, s.chainA.SenderAccount.GetAddress().String())
	s.Require().Error(err)
}

func (s *InterchainAccountsTestSuite) TestClosedChannelReopensWithMsgServer() {
	path := NewICAPath(s.chainA, s.chainB)
	s.coordinator.SetupConnections(path)

	err := SetupICAPath(path, s.chainA.SenderAccount.GetAddress().String())
	s.Require().NoError(err)

	// set the channel state to closed
	err = path.EndpointA.SetChannelState(channeltypes.CLOSED)
	s.Require().NoError(err)
	err = path.EndpointB.SetChannelState(channeltypes.CLOSED)
	s.Require().NoError(err)

	// reset endpoint channel ids
	path.EndpointA.ChannelID = ""
	path.EndpointB.ChannelID = ""

	// fetch the next channel sequence before reinitiating the channel handshake
	channelSeq := s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(s.chainA.GetContext())

	// route a new MsgRegisterInterchainAccount in order to reopen the
	msgServer := controllerkeeper.NewMsgServerImpl(&s.chainA.GetSimApp().ICAControllerKeeper)
	msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, s.chainA.SenderAccount.GetAddress().String(), path.EndpointA.ChannelConfig.Version)

	res, err := msgServer.RegisterInterchainAccount(s.chainA.GetContext(), msgRegisterInterchainAccount)
	s.Require().NoError(err)
	s.Require().Equal(channeltypes.FormatChannelIdentifier(channelSeq), res.ChannelId)

	// assign the channel sequence to endpointA before generating proofs and initiating the TRY step
	path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(channelSeq)

	path.EndpointA.Chain.NextBlock()

	err = path.EndpointB.ChanOpenTry()
	s.Require().NoError(err)

	err = path.EndpointA.ChanOpenAck()
	s.Require().NoError(err)

	err = path.EndpointB.ChanOpenConfirm()
	s.Require().NoError(err)
}
