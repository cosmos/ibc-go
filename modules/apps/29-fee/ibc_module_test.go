package fee_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	fee "github.com/cosmos/ibc-go/v3/modules/apps/29-fee"
	"github.com/cosmos/ibc-go/v3/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"
	"github.com/cosmos/ibc-go/v3/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	ibcmock "github.com/cosmos/ibc-go/v3/testing/mock"
)

var (
	defaultRecvFee    = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	defaultAckFee     = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(200)}}
	defaultTimeoutFee = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(300)}}
)

// Tests OnChanOpenInit on ChainA
func (suite *FeeTestSuite) TestOnChanOpenInit() {
	testCases := []struct {
		name    string
		version string
		expPass bool
	}{
		{
			"success - valid fee middleware and mock version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version})),
			true,
		},
		{
			"success - fee version not included, only perform mock logic",
			ibcmock.Version,
			true,
		},
		{
			"invalid fee middleware version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: "invalid-ics29-1", AppVersion: ibcmock.Version})),
			false,
		},
		{
			"invalid mock version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: "invalid-mock-version"})),
			false,
		},
		{
			"mock version not wrapped",
			types.Version,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()
			suite.coordinator.SetupConnections(suite.path)

			// setup mock callback
			suite.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
				portID, channelID string, chanCap *capabilitytypes.Capability,
				counterparty channeltypes.Counterparty, version string,
			) error {
				if version != ibcmock.Version {
					return fmt.Errorf("incorrect mock version")
				}
				return nil
			}

			suite.path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty := channeltypes.NewCounterparty(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)
			channel := &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{suite.path.EndpointA.ConnectionID},
				Version:        tc.version,
			}

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			chanCap, err := suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanOpenInit(suite.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, chanCap, counterparty, channel.Version)

			if tc.expPass {
				suite.Require().NoError(err, "unexpected error from version: %s", tc.version)
			} else {
				suite.Require().Error(err, "error not returned for version: %s", tc.version)
			}
		})
	}
}

// Tests OnChanOpenTry on ChainA
func (suite *FeeTestSuite) TestOnChanOpenTry() {
	testCases := []struct {
		name      string
		cpVersion string
		crossing  bool
		expPass   bool
	}{
		{
			"success - valid fee middleware version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version})),
			false,
			true,
		},
		{
			"success - valid mock version",
			ibcmock.Version,
			false,
			true,
		},
		{
			"success - crossing hellos: valid fee middleware",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version})),
			true,
			true,
		},
		{
			"invalid fee middleware version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: "invalid-ics29-1", AppVersion: ibcmock.Version})),
			false,
			false,
		},
		{
			"invalid mock version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: "invalid-mock-version"})),
			false,
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()
			suite.coordinator.SetupConnections(suite.path)
			suite.path.EndpointB.ChanOpenInit()

			// setup mock callback
			suite.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanOpenTry = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
				portID, channelID string, chanCap *capabilitytypes.Capability,
				counterparty channeltypes.Counterparty, counterpartyVersion string,
			) (string, error) {
				if counterpartyVersion != ibcmock.Version {
					return "", fmt.Errorf("incorrect mock version")
				}
				return ibcmock.Version, nil
			}

			var (
				chanCap *capabilitytypes.Capability
				ok      bool
				err     error
			)
			if tc.crossing {
				suite.path.EndpointA.ChanOpenInit()
				chanCap, ok = suite.chainA.GetSimApp().ScopedFeeMockKeeper.GetCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
				suite.Require().True(ok)
			} else {
				chanCap, err = suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID))
				suite.Require().NoError(err)
			}

			suite.path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty := channeltypes.NewCounterparty(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)
			channel := &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{suite.path.EndpointA.ConnectionID},
				Version:        tc.cpVersion,
			}

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			_, err = cbs.OnChanOpenTry(suite.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, chanCap, counterparty, tc.cpVersion)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// Tests OnChanOpenAck on ChainA
func (suite *FeeTestSuite) TestOnChanOpenAck() {
	testCases := []struct {
		name      string
		cpVersion string
		malleate  func(suite *FeeTestSuite)
		expPass   bool
	}{
		{
			"success",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version})),
			func(suite *FeeTestSuite) {},
			true,
		},
		{
			"invalid fee version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: "invalid-ics29-1", AppVersion: ibcmock.Version})),
			func(suite *FeeTestSuite) {},
			false,
		},
		{
			"invalid mock version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: "invalid-mock-version"})),
			func(suite *FeeTestSuite) {},
			false,
		},
		{
			"invalid version fails to unmarshal metadata",
			"invalid-version",
			func(suite *FeeTestSuite) {},
			false,
		},
		{
			"previous INIT set without fee, however counterparty set fee version", // note this can only happen with incompetent or malicious counterparty chain
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version})),
			func(suite *FeeTestSuite) {
				// do the first steps without fee version, then pass the fee version as counterparty version in ChanOpenACK
				suite.path.EndpointA.ChannelConfig.Version = ibcmock.Version
				suite.path.EndpointB.ChannelConfig.Version = ibcmock.Version
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.SetupConnections(suite.path)

			// setup mock callback
			suite.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanOpenAck = func(
				ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string,
			) error {
				if counterpartyVersion != ibcmock.Version {
					return fmt.Errorf("incorrect mock version")
				}
				return nil
			}

			// malleate test case
			tc.malleate(suite)

			suite.path.EndpointA.ChanOpenInit()
			suite.path.EndpointB.ChanOpenTry()

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanOpenAck(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, suite.path.EndpointA.Counterparty.ChannelID, tc.cpVersion)
			if tc.expPass {
				suite.Require().NoError(err, "unexpected error for case: %s", tc.name)
			} else {
				suite.Require().Error(err, "%s expected error but returned none", tc.name)
			}
		})
	}
}

func (suite *FeeTestSuite) TestOnChanCloseInit() {
	var (
		refundAcc sdk.AccAddress
		fee       types.Fee
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
			"application callback fails", func() {
				suite.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanCloseInit = func(
					ctx sdk.Context, portID, channelID string,
				) error {
					return fmt.Errorf("application callback fails")
				}
			}, false,
		},
		{
			"RefundFeesOnChannelClosure fails - invalid refund address", func() {
				// store the fee in state & update escrow account balance
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, "invalid refund address", nil)})

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				suite.Require().NoError(err)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path) // setup channel

			packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
			fee = types.Fee{
				RecvFee:    defaultRecvFee,
				AckFee:     defaultAckFee,
				TimeoutFee: defaultTimeoutFee,
			}

			refundAcc = suite.chainA.SenderAccount.GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
			err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
			suite.Require().NoError(err)

			tc.malleate()

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanCloseInit(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

// Tests OnChanCloseConfirm on chainA
func (suite *FeeTestSuite) TestOnChanCloseConfirm() {
	var (
		refundAcc sdk.AccAddress
		fee       types.Fee
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
			"application callback fails", func() {
				suite.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanCloseConfirm = func(
					ctx sdk.Context, portID, channelID string,
				) error {
					return fmt.Errorf("application callback fails")
				}
			}, false,
		},
		{
			"RefundChannelFeesOnClosure fails - refund address is invalid", func() {
				// store the fee in state & update escrow account balance
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, "invalid refund address", nil)})

				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, packetFees)
				err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				suite.Require().NoError(err)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path) // setup channel

			packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, 1)
			fee = types.Fee{
				RecvFee:    defaultRecvFee,
				AckFee:     defaultAckFee,
				TimeoutFee: defaultTimeoutFee,
			}

			refundAcc = suite.chainA.SenderAccount.GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
			err := suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
			suite.Require().NoError(err)

			tc.malleate()

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanCloseConfirm(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

		})
	}
}

func (suite *FeeTestSuite) TestOnRecvPacket() {
	testCases := []struct {
		name     string
		malleate func()
		// forwardRelayer bool indicates if there is a forwardRelayer address set
		forwardRelayer bool
		feeEnabled     bool
	}{
		{
			"success",
			func() {},
			true,
			true,
		},
		{
			"async write acknowledgement: ack is nil",
			func() {
				// setup mock callback
				suite.chainB.GetSimApp().FeeMockModule.IBCApp.OnRecvPacket = func(
					ctx sdk.Context,
					packet channeltypes.Packet,
					relayer sdk.AccAddress,
				) exported.Acknowledgement {
					return nil
				}
			},
			true,
			true,
		},
		{
			"fee not enabled",
			func() {
				suite.chainB.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainB.GetContext(), suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)
			},
			true,
			false,
		},
		{
			"forward address is not found",
			func() {
				suite.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainB.GetContext(), suite.chainA.SenderAccount.GetAddress().String(), "", suite.path.EndpointB.ChannelID)
			},
			false,
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			// setup pathAToC (chainA -> chainC) first in order to have different channel IDs for chainA & chainB
			suite.coordinator.Setup(suite.pathAToC)
			// setup path for chainA -> chainB
			suite.coordinator.Setup(suite.path)

			suite.chainB.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainB.GetContext(), suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)

			packet := suite.CreateMockPacket()

			// set up module and callbacks
			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			suite.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainB.GetContext(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), suite.path.EndpointB.ChannelID)

			// malleate test case
			tc.malleate()

			result := cbs.OnRecvPacket(suite.chainB.GetContext(), packet, suite.chainA.SenderAccount.GetAddress())

			switch {
			case tc.name == "success":
				forwardAddr, _ := suite.chainB.GetSimApp().IBCFeeKeeper.GetCounterpartyAddress(suite.chainB.GetContext(), suite.chainA.SenderAccount.GetAddress().String(), suite.path.EndpointB.ChannelID)

				expectedAck := types.IncentivizedAcknowledgement{
					Result:                ibcmock.MockAcknowledgement.Acknowledgement(),
					ForwardRelayerAddress: forwardAddr,
					UnderlyingAppSuccess:  true,
				}
				suite.Require().Equal(expectedAck, result)

			case !tc.feeEnabled:
				suite.Require().Equal(ibcmock.MockAcknowledgement, result)

			case tc.forwardRelayer && result == nil:
				suite.Require().Equal(nil, result)
				packetID := channeltypes.NewPacketId(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				// retrieve the forward relayer that was stored in `onRecvPacket`
				relayer, _ := suite.chainB.GetSimApp().IBCFeeKeeper.GetRelayerAddressForAsyncAck(suite.chainB.GetContext(), packetID)
				suite.Require().Equal(relayer, suite.chainA.SenderAccount.GetAddress().String())

			case !tc.forwardRelayer:
				expectedAck := types.IncentivizedAcknowledgement{
					Result:                ibcmock.MockAcknowledgement.Acknowledgement(),
					ForwardRelayerAddress: "",
					UnderlyingAppSuccess:  true,
				}
				suite.Require().Equal(expectedAck, result)
			}
		})
	}
}

// different channel than sending chain
func (suite *FeeTestSuite) TestOnAcknowledgementPacket() {
	var (
		ack                    []byte
		packetFee              types.PacketFee
		originalBalance        sdk.Coins
		expectedBalance        sdk.Coins
		expectedRelayerBalance sdk.Coins
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {
				expectedRelayerBalance = packetFee.Fee.RecvFee.Add(packetFee.Fee.AckFee[0])
			},
			true,
		},
		{
			"no op success without a packet fee",
			func() {
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, suite.chainA.SenderAccount.GetSequence())
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeesInEscrow(suite.chainA.GetContext(), packetID)

				ack = types.IncentivizedAcknowledgement{
					Result:                ibcmock.MockAcknowledgement.Acknowledgement(),
					ForwardRelayerAddress: suite.chainA.SenderAccount.GetAddress().String(),
				}.Acknowledgement()

				expectedBalance = originalBalance
			},
			true,
		},
		{
			"ack wrong format",
			func() {
				ack = []byte("unsupported acknowledgement format")

				expectedBalance = originalBalance
			},
			false,
		},
		{
			"channel is not fee not enabled, success",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
				ack = ibcmock.MockAcknowledgement.Acknowledgement()

				expectedBalance = originalBalance
			},
			true,
		},
		{
			"success: fee module is disabled, skip fee logic",
			func() {
				lockFeeModule(suite.chainA)

				expectedBalance = originalBalance
			},
			true,
		},
		{
			"fail on distribute receive fee (blocked address)",
			func() {
				blockedAddr := suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress()

				ack = types.IncentivizedAcknowledgement{
					Result:                ibcmock.MockAcknowledgement.Acknowledgement(),
					ForwardRelayerAddress: blockedAddr.String(),
				}.Acknowledgement()

				expectedRelayerBalance = packetFee.Fee.AckFee
				expectedBalance = expectedBalance.Add(packetFee.Fee.RecvFee...)
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path)
			packet := suite.CreateMockPacket()

			expectedRelayerBalance = sdk.Coins{} // reset

			// set up module and callbacks
			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			// escrow the packet fee
			packetID := channeltypes.NewPacketId(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			packetFee = types.NewPacketFee(
				types.Fee{
					RecvFee:    defaultRecvFee,
					AckFee:     defaultAckFee,
					TimeoutFee: defaultTimeoutFee,
				},
				suite.chainA.SenderAccount.GetAddress().String(),
				[]string{},
			)

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
			err = suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), types.ModuleName, packetFee.Fee.Total())
			suite.Require().NoError(err)

			relayerAddr := suite.chainB.SenderAccount.GetAddress()

			// must be changed explicitly
			ack = types.IncentivizedAcknowledgement{
				Result:                ibcmock.MockAcknowledgement.Acknowledgement(),
				ForwardRelayerAddress: relayerAddr.String(),
			}.Acknowledgement()

			// log original sender balance
			// NOTE: balance is logged after escrowing tokens
			originalBalance = sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))

			// default to success case
			expectedBalance = originalBalance.Add(packetFee.Fee.TimeoutFee[0])

			// malleate test case
			tc.malleate()

			err = cbs.OnAcknowledgementPacket(suite.chainA.GetContext(), packet, ack, relayerAddr)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			suite.Require().Equal(
				expectedBalance,
				sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom)),
			)

			relayerBalance := sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), relayerAddr, ibctesting.TestCoin.Denom))
			suite.Require().Equal(
				expectedRelayerBalance,
				relayerBalance,
			)
		})
	}
}

func (suite *FeeTestSuite) TestOnTimeoutPacket() {
	var (
		relayerAddr     sdk.AccAddress
		packetFee       types.PacketFee
		originalBalance sdk.Coins
		expectedBalance sdk.Coins
	)
	testCases := []struct {
		name              string
		malleate          func()
		expFeeDistributed bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"fee not enabled",
			func() {
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)

				expectedBalance = originalBalance
			},
			false,
		},
		{
			"fee module is disabled, skip fee logic",
			func() {
				lockFeeModule(suite.chainA)

				expectedBalance = originalBalance
			},
			false,
		},
		{
			"no op if identified packet fee doesn't exist",
			func() {
				// delete packet fee
				packetID := channeltypes.NewPacketId(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, suite.chainA.SenderAccount.GetSequence())
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeesInEscrow(suite.chainA.GetContext(), packetID)

				expectedBalance = originalBalance
			},
			false,
		},
		{
			"distribute fee fails for timeout fee (blocked address)",
			func() {
				relayerAddr = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), transfertypes.ModuleName).GetAddress()

				expectedBalance = originalBalance.
					Add(packetFee.Fee.RecvFee...).
					Add(packetFee.Fee.AckFee...).
					Add(packetFee.Fee.TimeoutFee...)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path)
			packet := suite.CreateMockPacket()

			// set up module and callbacks
			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			packetID := channeltypes.NewPacketId(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())

			// must be explicitly changed
			relayerAddr = suite.chainB.SenderAccount.GetAddress()

			packetFee = types.NewPacketFee(
				types.Fee{
					RecvFee:    defaultRecvFee,
					AckFee:     defaultAckFee,
					TimeoutFee: defaultTimeoutFee,
				},
				suite.chainA.SenderAccount.GetAddress().String(),
				[]string{},
			)

			suite.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(suite.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
			err = suite.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), types.ModuleName, packetFee.Fee.Total())
			suite.Require().NoError(err)

			// log original sender balance
			// NOTE: balance is logged after escrowing tokens
			originalBalance = sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))

			// default to success case
			expectedBalance = originalBalance.
				Add(packetFee.Fee.RecvFee[0]).
				Add(packetFee.Fee.AckFee[0])

			// malleate test case
			tc.malleate()

			err = cbs.OnTimeoutPacket(suite.chainA.GetContext(), packet, relayerAddr)
			suite.Require().NoError(err)

			suite.Require().Equal(
				expectedBalance,
				sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom)),
			)

			relayerBalance := sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), relayerAddr, ibctesting.TestCoin.Denom))
			if tc.expFeeDistributed {
				// there should no longer be a fee in escrow for this packet
				found := suite.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(suite.chainA.GetContext(), packetID)
				suite.Require().False(found)

				suite.Require().Equal(packetFee.Fee.TimeoutFee, relayerBalance)
			} else {
				suite.Require().Empty(relayerBalance)
			}
		})
	}
}

func (suite *FeeTestSuite) TestGetAppVersion() {
	var (
		portID        string
		channelID     string
		expAppVersion string
	)
	testCases := []struct {
		name     string
		malleate func()
		expFound bool
	}{
		{
			"success for fee enabled channel",
			func() {
				expAppVersion = ibcmock.Version
			},
			true,
		},
		{
			"success for non fee enabled channel",
			func() {
				path := ibctesting.NewPath(suite.chainA, suite.chainB)
				path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
				path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
				// by default a new path uses a non fee channel
				suite.coordinator.Setup(path)
				portID = path.EndpointA.ChannelConfig.PortID
				channelID = path.EndpointA.ChannelID

				expAppVersion = ibcmock.Version
			},
			true,
		},
		{
			"channel does not exist",
			func() {
				channelID = "does not exist"
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path)

			portID = suite.path.EndpointA.ChannelConfig.PortID
			channelID = suite.path.EndpointA.ChannelID

			// malleate test case
			tc.malleate()

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.MockFeePort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			feeModule := cbs.(fee.IBCModule)

			appVersion, found := feeModule.GetAppVersion(suite.chainA.GetContext(), portID, channelID)

			if tc.expFound {
				suite.Require().True(found)
				suite.Require().Equal(expAppVersion, appVersion)
			} else {
				suite.Require().False(found)
				suite.Require().Empty(appVersion)
			}
		})
	}
}
