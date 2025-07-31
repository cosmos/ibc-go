package controller_test

import (
	"errors"
	"strconv"
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller"
	controllerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	"github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

const invalidVersion = "invalid|version"

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// TestPortID defines a reusable port identifier for testing purposes
	TestPortID, _ = icatypes.NewControllerPortID(TestOwnerAddress)

	// TestVersion defines a reusable interchainaccounts version string for testing purposes
	TestVersion = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)
)

type InterchainAccountsTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

func TestICATestSuite(t *testing.T) {
	testifysuite.Run(t, new(InterchainAccountsTestSuite))
}

func (s *InterchainAccountsTestSuite) SetupTest() {
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

func (s *InterchainAccountsTestSuite) TestSetUnderlyingApplication() {
	var (
		app porttypes.IBCModule
		mw  porttypes.Middleware
	)
	testCases := []struct {
		name     string
		malleate func()
		expPanic bool
	}{
		{
			"success", func() {}, false,
		},
		{
			"nil underlying app", func() {
				app = nil
			}, true,
		},
		{
			"app already set", func() {
				mw.SetUnderlyingApplication(&ibcmock.IBCModule{})
			}, true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset

			app = &ibcmock.IBCModule{}
			mw = controller.NewIBCMiddleware(s.chainA.GetSimApp().ICAControllerKeeper)

			tc.malleate() // malleate mutates test data

			if tc.expPanic {
				s.Require().Panics(func() {
					mw.SetUnderlyingApplication(app)
				})
			} else {
				s.Require().NotPanics(func() {
					mw.SetUnderlyingApplication(app)
				})
			}
		})
	}
}

func (s *InterchainAccountsTestSuite) TestSetICS4Wrapper() {
	var wrapper porttypes.ICS4Wrapper
	mw := controller.NewIBCMiddleware(s.chainA.GetSimApp().ICAControllerKeeper)
	testCases := []struct {
		name     string
		malleate func()
		expPanic bool
	}{
		{
			"success", func() {}, false,
		},
		{
			"nil ICS4Wrapper", func() {
				wrapper = nil
			}, true,
		},
	}
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest() // reset
			wrapper = s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper
			tc.malleate() // malleate mutates test data
			if tc.expPanic {
				s.Require().Panics(func() {
					mw.SetICS4Wrapper(wrapper)
				})
			} else {
				s.Require().NotPanics(func() {
					mw.SetICS4Wrapper(wrapper)
				})
			}
		})
	}
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
		expErr   error
	}{
		{
			"success", func() {}, nil,
		},
		{
			"ICA auth module modification of channel version is ignored", func() {
				// NOTE: explicitly modify the channel version via the auth module callback,
				// ensuring the expected JSON encoded metadata is not modified upon return
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string,
					counterparty channeltypes.Counterparty, version string,
				) (string, error) {
					return "invalid-version", nil
				}
			}, nil,
		},
		{
			"controller submodule disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			}, types.ErrControllerSubModuleDisabled,
		},
		{
			"ICA auth module callback fails", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string,
					counterparty channeltypes.Counterparty, version string,
				) (string, error) {
					return "", errors.New("mock ica auth fails")
				}
			}, errors.New("mock ica auth fails"),
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, nil,
		},
		{
			"middleware disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string,
					counterparty channeltypes.Counterparty, version string,
				) (string, error) {
					return "", errors.New("error should be unreachable")
				}
			}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				// mock init interchain account
				portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
				s.Require().NoError(err)

				path.EndpointA.ChannelConfig.PortID = portID
				path.EndpointA.ChannelID = ibctesting.FirstChannelID

				s.chainA.GetSimApp().ICAControllerKeeper.SetMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				// default values
				counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
				channel = &channeltypes.Channel{
					State:          channeltypes.INIT,
					Ordering:       ordering,
					Counterparty:   counterparty,
					ConnectionHops: []string{path.EndpointA.ConnectionID},
					Version:        path.EndpointA.ChannelConfig.Version,
				}

				tc.malleate() // malleate mutates test data

				// ensure channel on chainA is set in state
				s.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, *channel)

				cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				s.Require().True(ok)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(s.chainA.GetSimApp().ICAControllerKeeper)
				}

				version, err := cbs.OnChanOpenInit(s.chainA.GetContext(), channel.Ordering, channel.ConnectionHops,
					path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel.Counterparty, channel.Version,
				)

				if tc.expErr == nil {
					s.Require().Equal(TestVersion, version)
					s.Require().NoError(err)
				} else {
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

// Test initiating a ChanOpenTry using the controller chain instead of the host chain
// ChainA is the controller chain. ChainB creates a controller port as well,
// attempting to trick chainA.
// Sending a MsgChanOpenTry will never reach the application callback due to
// core IBC checks not passing, so a call to the application callback is also
// done directly.
func (s *InterchainAccountsTestSuite) TestChanOpenTry() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest() // reset
		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
		s.Require().NoError(err)

		// chainB also creates a controller port
		err = RegisterInterchainAccount(path.EndpointB, TestOwnerAddress)
		s.Require().NoError(err)

		err = path.EndpointA.UpdateClient()
		s.Require().NoError(err)

		channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		initProof, proofHeight := path.EndpointB.Chain.QueryProof(channelKey)

		// use chainA (controller) for ChanOpenTry
		msg := channeltypes.NewMsgChannelOpenTry(path.EndpointA.ChannelConfig.PortID, TestVersion, ordering, []string{path.EndpointA.ConnectionID}, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, TestVersion, initProof, proofHeight, icatypes.ModuleName)
		handler := s.chainA.GetSimApp().MsgServiceRouter().Handler(msg)
		_, err = handler(s.chainA.GetContext(), msg)

		s.Require().Error(err)

		cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointB.ChannelConfig.PortID)
		s.Require().True(ok)

		counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

		version, err := cbs.OnChanOpenTry(
			s.chainA.GetContext(), path.EndpointA.ChannelConfig.Order, []string{path.EndpointA.ConnectionID},
			path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
			counterparty, path.EndpointB.ChannelConfig.Version,
		)
		s.Require().Error(err)
		s.Require().Empty(version)
	}
}

func (s *InterchainAccountsTestSuite) TestOnChanOpenAck() {
	var path *ibctesting.Path

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success", func() {}, nil,
		},
		{
			"controller submodule disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			}, types.ErrControllerSubModuleDisabled,
		},
		{
			"ICA OnChanOpenACK fails - invalid version", func() {
				path.EndpointB.ChannelConfig.Version = invalidVersion
			}, ibcerrors.ErrInvalidType,
		},
		{
			"ICA auth module callback fails", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenAck = func(
					ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string,
				) error {
					return errors.New("mock ica auth fails")
				}
			}, errors.New("mock ica auth fails"),
		},
		{
			"middleware disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenAck = func(
					ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string,
				) error {
					return errors.New("error should be unreachable")
				}
			}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest() // reset

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
				s.Require().NoError(err)

				err = path.EndpointB.ChanOpenTry()
				s.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				s.Require().True(ok)

				err = cbs.OnChanOpenAck(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelID, path.EndpointB.ChannelConfig.Version)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

// Test initiating a ChanOpenConfirm using the controller chain instead of the host chain
// ChainA is the controller chain. ChainB is the host chain
// Sending a MsgChanOpenConfirm will never reach the application callback due to
// core IBC checks not passing, so a call to the application callback is also
// done directly.
func (s *InterchainAccountsTestSuite) TestChanOpenConfirm() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest() // reset
		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
		s.Require().NoError(err)

		err = path.EndpointB.ChanOpenTry()
		s.Require().NoError(err)

		// chainB maliciously sets channel to OPEN
		channel := channeltypes.NewChannel(channeltypes.OPEN, ordering, channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, TestVersion)
		s.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(s.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel)

		// commit state changes so proof can be created
		s.chainB.NextBlock()

		err = path.EndpointA.UpdateClient()
		s.Require().NoError(err)

		// query proof from ChainB
		channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		ackProof, proofHeight := path.EndpointB.Chain.QueryProof(channelKey)

		// use chainA (controller) for ChanOpenConfirm
		msg := channeltypes.NewMsgChannelOpenConfirm(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ackProof, proofHeight, icatypes.ModuleName)
		handler := s.chainA.GetSimApp().MsgServiceRouter().Handler(msg)
		_, err = handler(s.chainA.GetContext(), msg)

		s.Require().Error(err)

		cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
		s.Require().True(ok)

		err = cbs.OnChanOpenConfirm(
			s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		)
		s.Require().Error(err)
	}
}

// OnChanCloseInit on controller (chainA)
func (s *InterchainAccountsTestSuite) TestOnChanCloseInit() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest() // reset

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
		s.Require().True(ok)

		err = cbs.OnChanCloseInit(
			s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		)

		s.Require().Error(err)
	}
}

func (s *InterchainAccountsTestSuite) TestOnChanCloseConfirm() {
	var (
		path     *ibctesting.Path
		isNilApp bool
	)

	testCases := []struct {
		name     string
		malleate func()
		expErr   error
	}{
		{
			"success", func() {}, nil,
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				s.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				s.Require().True(ok)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(s.chainA.GetSimApp().ICAControllerKeeper)
				}

				err = cbs.OnChanCloseConfirm(
					s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().ErrorIs(err, tc.expErr)
				}
			})
		}
	}
}

func (s *InterchainAccountsTestSuite) TestOnRecvPacket() {
	testCases := []struct {
		name       string
		malleate   func()
		expSuccess bool
	}{
		{
			"ICA OnRecvPacket fails with ErrInvalidChannelFlow", func() {}, false,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.name, func() {
				s.SetupTest() // reset

				path := NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				s.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
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

				ctx := s.chainA.GetContext()
				ack := cbs.OnRecvPacket(ctx, path.EndpointA.GetChannel().Version, packet, nil)
				s.Require().Equal(tc.expSuccess, ack.Success())

				expectedEvents := sdk.Events{
					sdk.NewEvent(
						icatypes.EventTypePacket,
						sdk.NewAttribute(sdk.AttributeKeyModule, icatypes.ModuleName),
						sdk.NewAttribute(icatypes.AttributeKeyControllerChannelID, packet.GetDestChannel()),
						sdk.NewAttribute(icatypes.AttributeKeyAckSuccess, strconv.FormatBool(false)),
						sdk.NewAttribute(icatypes.AttributeKeyAckError, "cannot receive packet on controller chain: invalid message sent to channel end"),
					),
				}.ToABCIEvents()

				expectedEvents = sdk.MarkEventsToIndex(expectedEvents, map[string]struct{}{})
				ibctesting.AssertEvents(&s.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())
			})
		}
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
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"controller submodule disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			}, types.ErrControllerSubModuleDisabled,
		},
		{
			"ICA auth module callback fails", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnAcknowledgementPacket = func(
					ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress,
				) error {
					return errors.New("mock ica auth fails")
				}
			}, errors.New("mock ica auth fails"),
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, nil,
		},
		{
			"middleware disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnAcknowledgementPacket = func(
					ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress,
				) error {
					return errors.New("error should be unreachable")
				}
			}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.msg, func() {
				s.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

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

				cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				s.Require().True(ok)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(s.chainA.GetSimApp().ICAControllerKeeper)
				}

				err = cbs.OnAcknowledgementPacket(s.chainA.GetContext(), path.EndpointA.GetChannel().Version, packet, []byte("ack"), nil)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
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
		expErr   error
	}{
		{
			"success",
			func() {},
			nil,
		},
		{
			"controller submodule disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.SetParams(s.chainA.GetContext(), types.NewParams(false))
			}, types.ErrControllerSubModuleDisabled,
		},
		{
			"ICA auth module callback fails", func() {
				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnTimeoutPacket = func(
					ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress,
				) error {
					return errors.New("mock ica auth fails")
				}
			}, errors.New("mock ica auth fails"),
		},
		{
			"nil underlying app", func() {
				isNilApp = true
			}, nil,
		},
		{
			"middleware disabled", func() {
				s.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				s.chainA.GetSimApp().ICAAuthModule.IBCApp.OnTimeoutPacket = func(
					ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress,
				) error {
					return errors.New("error should be unreachable")
				}
			}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.msg, func() {
				s.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(s.chainA, s.chainB, ordering)
				path.SetupConnections()

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

				cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				s.Require().True(ok)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(s.chainA.GetSimApp().ICAControllerKeeper)
				}

				err = cbs.OnTimeoutPacket(s.chainA.GetContext(), path.EndpointA.GetChannel().Version, packet, nil)

				if tc.expErr == nil {
					s.Require().NoError(err)
				} else {
					s.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
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
	}{
		{
			"success",
			func() {},
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			s.Run(tc.msg, func() {
				// reset
				s.SetupTest()
				TestVersion = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)

				// Setup a new path from A(controller) -> B(host)
				pathAToB = NewICAPath(s.chainA, s.chainB, ordering)
				pathAToB.SetupConnections()

				err := SetupICAPath(pathAToB, TestOwnerAddress)
				s.Require().NoError(err)

				// Setup a new path from C(controller) -> B(host)
				pathCToB = NewICAPath(s.chainC, s.chainB, ordering)
				pathCToB.SetupConnections()

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
}

func (s *InterchainAccountsTestSuite) TestGetAppVersion() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest() // reset

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		cbs, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
		s.Require().True(ok)

		controllerStack, ok := cbs.(porttypes.ICS4Wrapper)
		s.Require().True(ok)

		appVersion, found := controllerStack.GetAppVersion(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		s.Require().True(found)
		s.Require().Equal(path.EndpointA.ChannelConfig.Version, appVersion)
	}
}

func (s *InterchainAccountsTestSuite) TestInFlightHandshakeRespectsGoAPICaller() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest() // reset

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		// initiate a channel handshake such that channel.State == INIT
		err := RegisterInterchainAccount(path.EndpointA, s.chainA.SenderAccount.GetAddress().String())
		s.Require().NoError(err)

		// attempt to start a second handshake via the controller msg server
		msgServer := controllerkeeper.NewMsgServerImpl(s.chainA.GetSimApp().ICAControllerKeeper)
		msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, s.chainA.SenderAccount.GetAddress().String(), TestVersion, ordering)

		res, err := msgServer.RegisterInterchainAccount(s.chainA.GetContext(), msgRegisterInterchainAccount)
		s.Require().Error(err)
		s.Require().Nil(res)
	}
}

func (s *InterchainAccountsTestSuite) TestInFlightHandshakeRespectsMsgServerCaller() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest() // reset

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		// initiate a channel handshake such that channel.State == INIT
		msgServer := controllerkeeper.NewMsgServerImpl(s.chainA.GetSimApp().ICAControllerKeeper)
		msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, s.chainA.SenderAccount.GetAddress().String(), TestVersion, ordering)

		res, err := msgServer.RegisterInterchainAccount(s.chainA.GetContext(), msgRegisterInterchainAccount)
		s.Require().NotNil(res)
		s.Require().NoError(err)

		// attempt to start a second handshake via the legacy Go API
		err = RegisterInterchainAccount(path.EndpointA, s.chainA.SenderAccount.GetAddress().String())
		s.Require().Error(err)
	}
}

func (s *InterchainAccountsTestSuite) TestClosedChannelReopensWithMsgServer() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest() // reset

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, s.chainA.SenderAccount.GetAddress().String())
		s.Require().NoError(err)

		// set the channel state to closed
		path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })

		// reset endpoint channel ids
		path.EndpointA.ChannelID = ""
		path.EndpointB.ChannelID = ""

		// fetch the next channel sequence before reinitiating the channel handshake
		channelSeq := s.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(s.chainA.GetContext())

		// route a new MsgRegisterInterchainAccount in order to reopen the
		msgServer := controllerkeeper.NewMsgServerImpl(s.chainA.GetSimApp().ICAControllerKeeper)
		msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, s.chainA.SenderAccount.GetAddress().String(), path.EndpointA.ChannelConfig.Version, ordering)

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
}

func (s *InterchainAccountsTestSuite) TestPacketDataUnmarshalerInterface() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		s.SetupTest() // reset

		path := NewICAPath(s.chainA, s.chainB, ordering)
		path.SetupConnections()
		err := SetupICAPath(path, TestOwnerAddress)
		s.Require().NoError(err)

		expPacketData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: []byte("data"),
			Memo: "",
		}

		controllerMiddleware := controller.NewIBCMiddleware(s.chainA.GetSimApp().ICAControllerKeeper)
		packetData, version, err := controllerMiddleware.UnmarshalPacketData(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, expPacketData.GetBytes())
		s.Require().NoError(err)
		s.Require().Equal(version, path.EndpointA.ChannelConfig.Version)
		s.Require().Equal(expPacketData, packetData)

		// test invalid packet data
		invalidPacketData := []byte("invalid packet data")
		// Context, port identifier and channel identifier are not used for controller.
		packetData, version, err = controllerMiddleware.UnmarshalPacketData(s.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, invalidPacketData)
		s.Require().Error(err)
		s.Require().Empty(version)
		s.Require().Nil(packetData)
	}
}
