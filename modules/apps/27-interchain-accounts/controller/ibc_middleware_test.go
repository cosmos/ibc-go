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

func (suite *InterchainAccountsTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
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

func (suite *InterchainAccountsTestSuite) TestOnChanOpenInit() {
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
				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
					portID, channelID string,
					counterparty channeltypes.Counterparty, version string,
				) (string, error) {
					return "invalid-version", nil
				}
			}, nil,
		},
		{
			"controller submodule disabled", func() {
				suite.chainA.GetSimApp().ICAControllerKeeper.SetParams(suite.chainA.GetContext(), types.NewParams(false))
			}, types.ErrControllerSubModuleDisabled,
		},
		{
			"ICA auth module callback fails", func() {
				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
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
				suite.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
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
			suite.Run(tc.name, func() {
				suite.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				// mock init interchain account
				portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
				suite.Require().NoError(err)

				path.EndpointA.ChannelConfig.PortID = portID
				path.EndpointA.ChannelID = ibctesting.FirstChannelID

				suite.chainA.GetSimApp().ICAControllerKeeper.SetMiddlewareEnabled(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

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
				suite.chainA.GetSimApp().IBCKeeper.ChannelKeeper.SetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, *channel)

				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(ok)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(suite.chainA.GetSimApp().ICAControllerKeeper)
				}

				version, err := cbs.OnChanOpenInit(suite.chainA.GetContext(), channel.Ordering, channel.ConnectionHops,
					path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, channel.Counterparty, channel.Version,
				)

				if tc.expErr == nil {
					suite.Require().Equal(TestVersion, version)
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorContains(err, tc.expErr.Error())
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
func (suite *InterchainAccountsTestSuite) TestChanOpenTry() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest() // reset
		path := NewICAPath(suite.chainA, suite.chainB, ordering)
		path.SetupConnections()

		err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
		suite.Require().NoError(err)

		// chainB also creates a controller port
		err = RegisterInterchainAccount(path.EndpointB, TestOwnerAddress)
		suite.Require().NoError(err)

		err = path.EndpointA.UpdateClient()
		suite.Require().NoError(err)

		channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		initProof, proofHeight := path.EndpointB.Chain.QueryProof(channelKey)

		// use chainA (controller) for ChanOpenTry
		msg := channeltypes.NewMsgChannelOpenTry(path.EndpointA.ChannelConfig.PortID, TestVersion, ordering, []string{path.EndpointA.ConnectionID}, path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, TestVersion, initProof, proofHeight, icatypes.ModuleName)
		handler := suite.chainA.GetSimApp().MsgServiceRouter().Handler(msg)
		_, err = handler(suite.chainA.GetContext(), msg)

		suite.Require().Error(err)

		cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointB.ChannelConfig.PortID)
		suite.Require().True(ok)

		counterparty := channeltypes.NewCounterparty(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)

		version, err := cbs.OnChanOpenTry(
			suite.chainA.GetContext(), path.EndpointA.ChannelConfig.Order, []string{path.EndpointA.ConnectionID},
			path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
			counterparty, path.EndpointB.ChannelConfig.Version,
		)
		suite.Require().Error(err)
		suite.Require().Equal("", version)
	}
}

func (suite *InterchainAccountsTestSuite) TestOnChanOpenAck() {
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
			"controller submodule disabled", func() {
				suite.chainA.GetSimApp().ICAControllerKeeper.SetParams(suite.chainA.GetContext(), types.NewParams(false))
			}, types.ErrControllerSubModuleDisabled,
		},
		{
			"ICA OnChanOpenACK fails - invalid version", func() {
				path.EndpointB.ChannelConfig.Version = invalidVersion
			}, ibcerrors.ErrInvalidType,
		},
		{
			"ICA auth module callback fails", func() {
				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenAck = func(
					ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string,
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
				suite.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnChanOpenAck = func(
					ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string,
				) error {
					return errors.New("error should be unreachable")
				}
			}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			suite.Run(tc.name, func() {
				suite.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
				suite.Require().NoError(err)

				err = path.EndpointB.ChanOpenTry()
				suite.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(ok)

				err = cbs.OnChanOpenAck(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, path.EndpointB.ChannelID, path.EndpointB.ChannelConfig.Version)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(suite.chainA.GetSimApp().ICAControllerKeeper)
				}

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorContains(err, tc.expErr.Error())
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
func (suite *InterchainAccountsTestSuite) TestChanOpenConfirm() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest() // reset
		path := NewICAPath(suite.chainA, suite.chainB, ordering)
		path.SetupConnections()

		err := RegisterInterchainAccount(path.EndpointA, TestOwnerAddress)
		suite.Require().NoError(err)

		err = path.EndpointB.ChanOpenTry()
		suite.Require().NoError(err)

		// chainB maliciously sets channel to OPEN
		channel := channeltypes.NewChannel(channeltypes.OPEN, ordering, channeltypes.NewCounterparty(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID), []string{path.EndpointB.ConnectionID}, TestVersion)
		suite.chainB.GetSimApp().GetIBCKeeper().ChannelKeeper.SetChannel(suite.chainB.GetContext(), path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, channel)

		// commit state changes so proof can be created
		suite.chainB.NextBlock()

		err = path.EndpointA.UpdateClient()
		suite.Require().NoError(err)

		// query proof from ChainB
		channelKey := host.ChannelKey(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID)
		ackProof, proofHeight := path.EndpointB.Chain.QueryProof(channelKey)

		// use chainA (controller) for ChanOpenConfirm
		msg := channeltypes.NewMsgChannelOpenConfirm(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, ackProof, proofHeight, icatypes.ModuleName)
		handler := suite.chainA.GetSimApp().MsgServiceRouter().Handler(msg)
		_, err = handler(suite.chainA.GetContext(), msg)

		suite.Require().Error(err)

		cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
		suite.Require().True(ok)

		err = cbs.OnChanOpenConfirm(
			suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		)
		suite.Require().Error(err)
	}
}

// OnChanCloseInit on controller (chainA)
func (suite *InterchainAccountsTestSuite) TestOnChanCloseInit() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest() // reset

		path := NewICAPath(suite.chainA, suite.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		suite.Require().NoError(err)

		cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
		suite.Require().True(ok)

		err = cbs.OnChanCloseInit(
			suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
		)

		suite.Require().Error(err)
	}
}

func (suite *InterchainAccountsTestSuite) TestOnChanCloseConfirm() {
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
			suite.Run(tc.name, func() {
				suite.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				suite.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(ok)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(suite.chainA.GetSimApp().ICAControllerKeeper)
				}

				err = cbs.OnChanCloseConfirm(
					suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorIs(err, tc.expErr)
				}
			})
		}
	}
}

func (suite *InterchainAccountsTestSuite) TestOnRecvPacket() {
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
			suite.Run(tc.name, func() {
				suite.SetupTest() // reset

				path := NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				suite.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(ok)

				packet := channeltypes.NewPacket(
					[]byte("empty packet data"),
					suite.chainB.SenderAccount.GetSequence(),
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					clienttypes.NewHeight(0, 100),
					0,
				)

				ctx := suite.chainA.GetContext()
				ack := cbs.OnRecvPacket(ctx, path.EndpointA.GetChannel().Version, packet, nil)
				suite.Require().Equal(tc.expSuccess, ack.Success())

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
				ibctesting.AssertEvents(&suite.Suite, expectedEvents, ctx.EventManager().Events().ToABCIEvents())
			})
		}
	}
}

func (suite *InterchainAccountsTestSuite) TestOnAcknowledgementPacket() {
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
				suite.chainA.GetSimApp().ICAControllerKeeper.SetParams(suite.chainA.GetContext(), types.NewParams(false))
			}, types.ErrControllerSubModuleDisabled,
		},
		{
			"ICA auth module callback fails", func() {
				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnAcknowledgementPacket = func(
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
				suite.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnAcknowledgementPacket = func(
					ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress,
				) error {
					return errors.New("error should be unreachable")
				}
			}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			suite.Run(tc.msg, func() {
				suite.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				suite.Require().NoError(err)

				packet := channeltypes.NewPacket(
					[]byte("empty packet data"),
					suite.chainA.SenderAccount.GetSequence(),
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					clienttypes.NewHeight(0, 100),
					0,
				)

				tc.malleate() // malleate mutates test data

				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(ok)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(suite.chainA.GetSimApp().ICAControllerKeeper)
				}

				err = cbs.OnAcknowledgementPacket(suite.chainA.GetContext(), path.EndpointA.GetChannel().Version, packet, []byte("ack"), nil)

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (suite *InterchainAccountsTestSuite) TestOnTimeoutPacket() {
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
				suite.chainA.GetSimApp().ICAControllerKeeper.SetParams(suite.chainA.GetContext(), types.NewParams(false))
			}, types.ErrControllerSubModuleDisabled,
		},
		{
			"ICA auth module callback fails", func() {
				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnTimeoutPacket = func(
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
				suite.chainA.GetSimApp().ICAControllerKeeper.DeleteMiddlewareEnabled(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ConnectionID)

				suite.chainA.GetSimApp().ICAAuthModule.IBCApp.OnTimeoutPacket = func(
					ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress,
				) error {
					return errors.New("error should be unreachable")
				}
			}, nil,
		},
	}

	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		for _, tc := range testCases {
			suite.Run(tc.msg, func() {
				suite.SetupTest() // reset
				isNilApp = false

				path = NewICAPath(suite.chainA, suite.chainB, ordering)
				path.SetupConnections()

				err := SetupICAPath(path, TestOwnerAddress)
				suite.Require().NoError(err)

				packet := channeltypes.NewPacket(
					[]byte("empty packet data"),
					suite.chainA.SenderAccount.GetSequence(),
					path.EndpointA.ChannelConfig.PortID,
					path.EndpointA.ChannelID,
					path.EndpointB.ChannelConfig.PortID,
					path.EndpointB.ChannelID,
					clienttypes.NewHeight(0, 100),
					0,
				)

				tc.malleate() // malleate mutates test data

				cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(ok)

				if isNilApp {
					cbs = controller.NewIBCMiddleware(suite.chainA.GetSimApp().ICAControllerKeeper)
				}

				err = cbs.OnTimeoutPacket(suite.chainA.GetContext(), path.EndpointA.GetChannel().Version, packet, nil)

				if tc.expErr == nil {
					suite.Require().NoError(err)
				} else {
					suite.Require().ErrorContains(err, tc.expErr.Error())
				}
			})
		}
	}
}

func (suite *InterchainAccountsTestSuite) TestSingleHostMultipleControllers() {
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
			suite.Run(tc.msg, func() {
				// reset
				suite.SetupTest()
				TestVersion = icatypes.NewDefaultMetadataString(ibctesting.FirstConnectionID, ibctesting.FirstConnectionID)

				// Setup a new path from A(controller) -> B(host)
				pathAToB = NewICAPath(suite.chainA, suite.chainB, ordering)
				pathAToB.SetupConnections()

				err := SetupICAPath(pathAToB, TestOwnerAddress)
				suite.Require().NoError(err)

				// Setup a new path from C(controller) -> B(host)
				pathCToB = NewICAPath(suite.chainC, suite.chainB, ordering)
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
				suite.Require().NoError(err)

				tc.malleate() // malleate mutates test data

				accAddressChainA, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), pathAToB.EndpointB.ConnectionID, pathAToB.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				accAddressChainC, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), pathCToB.EndpointB.ConnectionID, pathCToB.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				suite.Require().NotEqual(accAddressChainA, accAddressChainC)

				chainAChannelID, found := suite.chainB.GetSimApp().ICAHostKeeper.GetActiveChannelID(suite.chainB.GetContext(), pathAToB.EndpointB.ConnectionID, pathAToB.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				chainCChannelID, found := suite.chainB.GetSimApp().ICAHostKeeper.GetActiveChannelID(suite.chainB.GetContext(), pathCToB.EndpointB.ConnectionID, pathCToB.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				suite.Require().NotEqual(chainAChannelID, chainCChannelID)
			})
		}
	}
}

func (suite *InterchainAccountsTestSuite) TestGetAppVersion() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest() // reset

		path := NewICAPath(suite.chainA, suite.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, TestOwnerAddress)
		suite.Require().NoError(err)

		cbs, ok := suite.chainA.App.GetIBCKeeper().PortKeeper.Route(path.EndpointA.ChannelConfig.PortID)
		suite.Require().True(ok)

		controllerStack, ok := cbs.(porttypes.ICS4Wrapper)
		suite.Require().True(ok)

		appVersion, found := controllerStack.GetAppVersion(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
		suite.Require().True(found)
		suite.Require().Equal(path.EndpointA.ChannelConfig.Version, appVersion)
	}
}

func (suite *InterchainAccountsTestSuite) TestInFlightHandshakeRespectsGoAPICaller() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest() // reset

		path := NewICAPath(suite.chainA, suite.chainB, ordering)
		path.SetupConnections()

		// initiate a channel handshake such that channel.State == INIT
		err := RegisterInterchainAccount(path.EndpointA, suite.chainA.SenderAccount.GetAddress().String())
		suite.Require().NoError(err)

		// attempt to start a second handshake via the controller msg server
		msgServer := controllerkeeper.NewMsgServerImpl(suite.chainA.GetSimApp().ICAControllerKeeper)
		msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, suite.chainA.SenderAccount.GetAddress().String(), TestVersion, ordering)

		res, err := msgServer.RegisterInterchainAccount(suite.chainA.GetContext(), msgRegisterInterchainAccount)
		suite.Require().Error(err)
		suite.Require().Nil(res)
	}
}

func (suite *InterchainAccountsTestSuite) TestInFlightHandshakeRespectsMsgServerCaller() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest() // reset

		path := NewICAPath(suite.chainA, suite.chainB, ordering)
		path.SetupConnections()

		// initiate a channel handshake such that channel.State == INIT
		msgServer := controllerkeeper.NewMsgServerImpl(suite.chainA.GetSimApp().ICAControllerKeeper)
		msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, suite.chainA.SenderAccount.GetAddress().String(), TestVersion, ordering)

		res, err := msgServer.RegisterInterchainAccount(suite.chainA.GetContext(), msgRegisterInterchainAccount)
		suite.Require().NotNil(res)
		suite.Require().NoError(err)

		// attempt to start a second handshake via the legacy Go API
		err = RegisterInterchainAccount(path.EndpointA, suite.chainA.SenderAccount.GetAddress().String())
		suite.Require().Error(err)
	}
}

func (suite *InterchainAccountsTestSuite) TestClosedChannelReopensWithMsgServer() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest() // reset

		path := NewICAPath(suite.chainA, suite.chainB, ordering)
		path.SetupConnections()

		err := SetupICAPath(path, suite.chainA.SenderAccount.GetAddress().String())
		suite.Require().NoError(err)

		// set the channel state to closed
		path.EndpointA.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })
		path.EndpointB.UpdateChannel(func(channel *channeltypes.Channel) { channel.State = channeltypes.CLOSED })

		// reset endpoint channel ids
		path.EndpointA.ChannelID = ""
		path.EndpointB.ChannelID = ""

		// fetch the next channel sequence before reinitiating the channel handshake
		channelSeq := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(suite.chainA.GetContext())

		// route a new MsgRegisterInterchainAccount in order to reopen the
		msgServer := controllerkeeper.NewMsgServerImpl(suite.chainA.GetSimApp().ICAControllerKeeper)
		msgRegisterInterchainAccount := types.NewMsgRegisterInterchainAccount(path.EndpointA.ConnectionID, suite.chainA.SenderAccount.GetAddress().String(), path.EndpointA.ChannelConfig.Version, ordering)

		res, err := msgServer.RegisterInterchainAccount(suite.chainA.GetContext(), msgRegisterInterchainAccount)
		suite.Require().NoError(err)
		suite.Require().Equal(channeltypes.FormatChannelIdentifier(channelSeq), res.ChannelId)

		// assign the channel sequence to endpointA before generating proofs and initiating the TRY step
		path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(channelSeq)

		path.EndpointA.Chain.NextBlock()

		err = path.EndpointB.ChanOpenTry()
		suite.Require().NoError(err)

		err = path.EndpointA.ChanOpenAck()
		suite.Require().NoError(err)

		err = path.EndpointB.ChanOpenConfirm()
		suite.Require().NoError(err)
	}
}

func (suite *InterchainAccountsTestSuite) TestPacketDataUnmarshalerInterface() {
	for _, ordering := range []channeltypes.Order{channeltypes.UNORDERED, channeltypes.ORDERED} {
		suite.SetupTest() // reset

		path := NewICAPath(suite.chainA, suite.chainB, ordering)
		path.SetupConnections()
		err := SetupICAPath(path, TestOwnerAddress)
		suite.Require().NoError(err)

		expPacketData := icatypes.InterchainAccountPacketData{
			Type: icatypes.EXECUTE_TX,
			Data: []byte("data"),
			Memo: "",
		}

		controllerMiddleware := controller.NewIBCMiddleware(suite.chainA.GetSimApp().ICAControllerKeeper)
		packetData, version, err := controllerMiddleware.UnmarshalPacketData(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, expPacketData.GetBytes())
		suite.Require().NoError(err)
		suite.Require().Equal(version, path.EndpointA.ChannelConfig.Version)
		suite.Require().Equal(expPacketData, packetData)

		// test invalid packet data
		invalidPacketData := []byte("invalid packet data")
		// Context, port identifier and channel identifier are not used for controller.
		packetData, version, err = controllerMiddleware.UnmarshalPacketData(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, invalidPacketData)
		suite.Require().Error(err)
		suite.Require().Empty(version)
		suite.Require().Nil(packetData)
	}
}
