package fee_test

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	fee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	"github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	ibcmock "github.com/cosmos/ibc-go/v7/testing/mock"
)

var (
	defaultRecvFee    = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(100)}}
	defaultAckFee     = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(200)}}
	defaultTimeoutFee = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(300)}}
	smallAmount       = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.NewInt(50)}}
)

// Tests OnChanOpenInit on ChainA
func (s *FeeTestSuite) TestOnChanOpenInit() {
	testCases := []struct {
		name         string
		version      string
		expPass      bool
		isFeeEnabled bool
	}{
		{
			"success - valid fee middleware and mock version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version})),
			true,
			true,
		},
		{
			"success - fee version not included, only perform mock logic",
			ibcmock.Version,
			true,
			false,
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
		{
			"mock version not wrapped",
			types.Version,
			false,
			false,
		},
		{
			"passing an empty string returns default version",
			"",
			true,
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			// reset suite
			s.SetupTest()
			s.coordinator.SetupConnections(s.path)

			// setup mock callback
			s.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanOpenInit = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
				portID, channelID string, chanCap *capabilitytypes.Capability,
				counterparty channeltypes.Counterparty, version string,
			) (string, error) {
				if version != ibcmock.Version {
					return "", fmt.Errorf("incorrect mock version")
				}
				return ibcmock.Version, nil
			}

			s.path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty := channeltypes.NewCounterparty(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)
			channel := &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{s.path.EndpointA.ConnectionID},
				Version:        tc.version,
			}

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			chanCap, err := s.chainA.App.GetScopedIBCKeeper().NewCapability(s.chainA.GetContext(), host.ChannelCapabilityPath(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID))
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			version, err := cbs.OnChanOpenInit(s.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, chanCap, counterparty, channel.Version)

			if tc.expPass {
				// check if the channel is fee enabled. If so version string should include metaData
				if tc.isFeeEnabled {
					versionMetadata := types.Metadata{
						FeeVersion: types.Version,
						AppVersion: ibcmock.Version,
					}

					versionBytes, err := types.ModuleCdc.MarshalJSON(&versionMetadata)
					s.Require().NoError(err)

					s.Require().Equal(version, string(versionBytes))
				} else {
					s.Require().Equal(ibcmock.Version, version)
				}

				s.Require().NoError(err, "unexpected error from version: %s", tc.version)
			} else {
				s.Require().Error(err, "error not returned for version: %s", tc.version)
				s.Require().Equal("", version)
			}
		})
	}
}

// Tests OnChanOpenTry on ChainA
func (s *FeeTestSuite) TestOnChanOpenTry() {
	testCases := []struct {
		name      string
		cpVersion string
		expPass   bool
	}{
		{
			"success - valid fee middleware version",
			string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: ibcmock.Version})),
			true,
		},
		{
			"success - valid mock version",
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
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			// reset suite
			s.SetupTest()
			s.coordinator.SetupConnections(s.path)
			err := s.path.EndpointB.ChanOpenInit()
			s.Require().NoError(err)

			// setup mock callback
			s.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanOpenTry = func(ctx sdk.Context, order channeltypes.Order, connectionHops []string,
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
			)

			chanCap, err = s.chainA.App.GetScopedIBCKeeper().NewCapability(s.chainA.GetContext(), host.ChannelCapabilityPath(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID))
			s.Require().NoError(err)

			s.path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty := channeltypes.NewCounterparty(s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)
			channel := &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{s.path.EndpointA.ConnectionID},
				Version:        tc.cpVersion,
			}

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			_, err = cbs.OnChanOpenTry(s.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, chanCap, counterparty, tc.cpVersion)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// Tests OnChanOpenAck on ChainA
func (s *FeeTestSuite) TestOnChanOpenAck() {
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
				s.path.EndpointA.ChannelConfig.Version = ibcmock.Version
				s.path.EndpointB.ChannelConfig.Version = ibcmock.Version
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.SetupConnections(s.path)

			// setup mock callback
			s.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanOpenAck = func(
				ctx sdk.Context, portID, channelID string, counterpartyChannelID string, counterpartyVersion string,
			) error {
				if counterpartyVersion != ibcmock.Version {
					return fmt.Errorf("incorrect mock version")
				}
				return nil
			}

			// malleate test case
			tc.malleate(s)

			err := s.path.EndpointA.ChanOpenInit()
			s.Require().NoError(err)
			err = s.path.EndpointB.ChanOpenTry()
			s.Require().NoError(err)

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			err = cbs.OnChanOpenAck(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, s.path.EndpointA.Counterparty.ChannelID, tc.cpVersion)
			if tc.expPass {
				s.Require().NoError(err, "unexpected error for case: %s", tc.name)
			} else {
				s.Require().Error(err, "%s expected error but returned none", tc.name)
			}
		})
	}
}

func (s *FeeTestSuite) TestOnChanCloseInit() {
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
				s.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanCloseInit = func(
					ctx sdk.Context, portID, channelID string,
				) error {
					return fmt.Errorf("application callback fails")
				}
			}, false,
		},
		{
			"RefundFeesOnChannelClosure continues - invalid refund address", func() {
				// store the fee in state & update escrow account balance
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, "invalid refund address", nil)})

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, packetFees)
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				s.Require().NoError(err)
			},
			true,
		},
		{
			"fee module locked", func() {
				lockFeeModule(s.chainA)
			},
			false,
		},
		{
			"fee module is not enabled", func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.Setup(s.path) // setup channel

			packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
			fee = types.Fee{
				RecvFee:    defaultRecvFee,
				AckFee:     defaultAckFee,
				TimeoutFee: defaultTimeoutFee,
			}

			refundAcc = s.chainA.SenderAccount.GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
			err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
			s.Require().NoError(err)

			tc.malleate()

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			err = cbs.OnChanCloseInit(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

// Tests OnChanCloseConfirm on chainA
func (s *FeeTestSuite) TestOnChanCloseConfirm() {
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
				s.chainA.GetSimApp().FeeMockModule.IBCApp.OnChanCloseConfirm = func(
					ctx sdk.Context, portID, channelID string,
				) error {
					return fmt.Errorf("application callback fails")
				}
			}, false,
		},
		{
			"RefundChannelFeesOnClosure continues - refund address is invalid", func() {
				// store the fee in state & update escrow account balance
				packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, uint64(1))
				packetFees := types.NewPacketFees([]types.PacketFee{types.NewPacketFee(fee, "invalid refund address", nil)})

				s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, packetFees)
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
				s.Require().NoError(err)
			},
			true,
		},
		{
			"fee module locked", func() {
				lockFeeModule(s.chainA)
			},
			false,
		},
		{
			"fee module is not enabled", func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.Setup(s.path) // setup channel

			packetID := channeltypes.NewPacketID(s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID, 1)
			fee = types.Fee{
				RecvFee:    defaultRecvFee,
				AckFee:     defaultAckFee,
				TimeoutFee: defaultTimeoutFee,
			}

			refundAcc = s.chainA.SenderAccount.GetAddress()
			packetFee := types.NewPacketFee(fee, refundAcc.String(), []string{})

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
			err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAcc, types.ModuleName, fee.Total())
			s.Require().NoError(err)

			tc.malleate()

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			err = cbs.OnChanCloseConfirm(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *FeeTestSuite) TestOnRecvPacket() {
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
				s.chainB.GetSimApp().FeeMockModule.IBCApp.OnRecvPacket = func(
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
				s.chainB.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainB.GetContext(), s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)
			},
			true,
			false,
		},
		{
			"forward address is not found",
			func() {
				s.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(s.chainB.GetContext(), s.chainA.SenderAccount.GetAddress().String(), "", s.path.EndpointB.ChannelID)
			},
			false,
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			// setup pathAToC (chainA -> chainC) first in order to have different channel IDs for chainA & chainB
			s.coordinator.Setup(s.pathAToC)
			// setup path for chainA -> chainB
			s.coordinator.Setup(s.path)

			s.chainB.GetSimApp().IBCFeeKeeper.SetFeeEnabled(s.chainB.GetContext(), s.path.EndpointB.ChannelConfig.PortID, s.path.EndpointB.ChannelID)

			packet := s.CreateMockPacket()

			// set up module and callbacks
			module, _, err := s.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainB.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			cbs, ok := s.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			s.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyPayeeAddress(s.chainB.GetContext(), s.chainA.SenderAccount.GetAddress().String(), s.chainB.SenderAccount.GetAddress().String(), s.path.EndpointB.ChannelID)

			// malleate test case
			tc.malleate()

			result := cbs.OnRecvPacket(s.chainB.GetContext(), packet, s.chainA.SenderAccount.GetAddress())

			switch {
			case tc.name == "success":
				forwardAddr, _ := s.chainB.GetSimApp().IBCFeeKeeper.GetCounterpartyPayeeAddress(s.chainB.GetContext(), s.chainA.SenderAccount.GetAddress().String(), s.path.EndpointB.ChannelID)

				expectedAck := types.IncentivizedAcknowledgement{
					AppAcknowledgement:    ibcmock.MockAcknowledgement.Acknowledgement(),
					ForwardRelayerAddress: forwardAddr,
					UnderlyingAppSuccess:  true,
				}
				s.Require().Equal(expectedAck, result)

			case !tc.feeEnabled:
				s.Require().Equal(ibcmock.MockAcknowledgement, result)

			case tc.forwardRelayer && result == nil:
				s.Require().Equal(nil, result)
				packetID := channeltypes.NewPacketID(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())

				// retrieve the forward relayer that was stored in `onRecvPacket`
				relayer, _ := s.chainB.GetSimApp().IBCFeeKeeper.GetRelayerAddressForAsyncAck(s.chainB.GetContext(), packetID)
				s.Require().Equal(relayer, s.chainA.SenderAccount.GetAddress().String())

			case !tc.forwardRelayer:
				expectedAck := types.IncentivizedAcknowledgement{
					AppAcknowledgement:    ibcmock.MockAcknowledgement.Acknowledgement(),
					ForwardRelayerAddress: "",
					UnderlyingAppSuccess:  true,
				}
				s.Require().Equal(expectedAck, result)
			}
		})
	}
}

func (s *FeeTestSuite) TestOnAcknowledgementPacket() {
	var (
		ack                 []byte
		packetID            channeltypes.PacketId
		packetFee           types.PacketFee
		refundAddr          sdk.AccAddress
		relayerAddr         sdk.AccAddress
		expRefundAccBalance sdk.Coins
		expPayeeAccBalance  sdk.Coins
	)

	testCases := []struct {
		name      string
		malleate  func()
		expPass   bool
		expResult func()
	}{
		{
			"success",
			func() {
				// retrieve the relayer acc balance and add the expected recv and ack fees
				relayerAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), relayerAddr, sdk.DefaultBondDenom))
				expPayeeAccBalance = relayerAccBalance.Add(packetFee.Fee.RecvFee...).Add(packetFee.Fee.AckFee...)

				// retrieve the refund acc balance and add the expected timeout fees
				refundAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom))
				expRefundAccBalance = refundAccBalance.Add(packetFee.Fee.TimeoutFee...)
			},
			true,
			func() {
				// assert that the packet fees have been distributed
				found := s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID)
				s.Require().False(found)

				relayerAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), relayerAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expPayeeAccBalance, sdk.NewCoins(relayerAccBalance))

				refundAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expRefundAccBalance, sdk.NewCoins(refundAccBalance))
			},
		},
		{
			"success: with registered payee address",
			func() {
				payeeAddr := s.chainA.SenderAccounts[2].SenderAccount.GetAddress()
				s.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
					s.chainA.GetContext(),
					s.chainA.SenderAccount.GetAddress().String(),
					payeeAddr.String(),
					s.path.EndpointA.ChannelID,
				)

				// reassign ack.ForwardRelayerAddress to the registered payee address
				ack = types.NewIncentivizedAcknowledgement(payeeAddr.String(), ibcmock.MockAcknowledgement.Acknowledgement(), true).Acknowledgement()

				// retrieve the payee acc balance and add the expected recv and ack fees
				payeeAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), payeeAddr, sdk.DefaultBondDenom))
				expPayeeAccBalance = payeeAccBalance.Add(packetFee.Fee.RecvFee...).Add(packetFee.Fee.AckFee...)

				// retrieve the refund acc balance and add the expected timeout fees
				refundAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom))
				expRefundAccBalance = refundAccBalance.Add(packetFee.Fee.TimeoutFee...)
			},
			true,
			func() {
				// assert that the packet fees have been distributed
				found := s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID)
				s.Require().False(found)

				payeeAddr := s.chainA.SenderAccounts[2].SenderAccount.GetAddress()
				payeeAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), payeeAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expPayeeAccBalance, sdk.NewCoins(payeeAccBalance))

				refundAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expRefundAccBalance, sdk.NewCoins(refundAccBalance))
			},
		},
		{
			"success: no op without a packet fee",
			func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeesInEscrow(s.chainA.GetContext(), packetID)

				ack = types.IncentivizedAcknowledgement{
					AppAcknowledgement:    ibcmock.MockAcknowledgement.Acknowledgement(),
					ForwardRelayerAddress: "",
				}.Acknowledgement()
			},
			true,
			func() {
				found := s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID)
				s.Require().False(found)
			},
		},
		{
			"success: channel is not fee enabled",
			func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
				ack = ibcmock.MockAcknowledgement.Acknowledgement()
			},
			true,
			func() {},
		},
		{
			"success: fee module is disabled, skip fee logic",
			func() {
				lockFeeModule(s.chainA)
			},
			true,
			func() {
				s.Require().Equal(true, s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(s.chainA.GetContext()))
			},
		},
		{
			"success: fail to distribute recv fee (blocked address), returned to refund account",
			func() {
				blockedAddr := s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress()

				// reassign ack.ForwardRelayerAddress to a blocked address
				ack = types.NewIncentivizedAcknowledgement(blockedAddr.String(), ibcmock.MockAcknowledgement.Acknowledgement(), true).Acknowledgement()

				// retrieve the relayer acc balance and add the expected ack fees
				relayerAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), relayerAddr, sdk.DefaultBondDenom))
				expPayeeAccBalance = relayerAccBalance.Add(packetFee.Fee.AckFee...)

				// retrieve the refund acc balance and add the expected recv fees and timeout fees
				refundAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom))
				expRefundAccBalance = refundAccBalance.Add(packetFee.Fee.RecvFee...).Add(packetFee.Fee.TimeoutFee...)
			},
			true,
			func() {
				// assert that the packet fees have been distributed
				found := s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID)
				s.Require().False(found)

				relayerAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), relayerAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expPayeeAccBalance, sdk.NewCoins(relayerAccBalance))

				refundAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expRefundAccBalance, sdk.NewCoins(refundAccBalance))
			},
		},
		{
			"fail: fee distribution fails and fee module is locked when escrow account does not have sufficient funds",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(s.chainA.GetContext(), types.ModuleName, s.chainA.SenderAccount.GetAddress(), smallAmount)
				s.Require().NoError(err)
			},
			true,
			func() {
				s.Require().Equal(true, s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(s.chainA.GetContext()))
			},
		},
		{
			"ack wrong format",
			func() {
				ack = []byte("unsupported acknowledgement format")
			},
			false,
			func() {},
		},
		{
			"invalid registered payee address",
			func() {
				payeeAddr := "invalid-address"
				s.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
					s.chainA.GetContext(),
					s.chainA.SenderAccount.GetAddress().String(),
					payeeAddr,
					s.path.EndpointA.ChannelID,
				)
			},
			false,
			func() {},
		},
		{
			"application callback fails",
			func() {
				s.chainA.GetSimApp().FeeMockModule.IBCApp.OnAcknowledgementPacket = func(_ sdk.Context, _ channeltypes.Packet, _ []byte, _ sdk.AccAddress) error {
					return fmt.Errorf("mock fee app callback fails")
				}
			},
			false,
			func() {},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.Setup(s.path)

			relayerAddr = s.chainA.SenderAccounts[0].SenderAccount.GetAddress()
			refundAddr = s.chainA.SenderAccounts[1].SenderAccount.GetAddress()

			packet := s.CreateMockPacket()
			packetID = channeltypes.NewPacketID(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			packetFee = types.NewPacketFee(types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee), refundAddr.String(), nil)

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))

			err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), refundAddr, types.ModuleName, packetFee.Fee.Total())
			s.Require().NoError(err)

			ack = types.NewIncentivizedAcknowledgement(relayerAddr.String(), ibcmock.MockAcknowledgement.Acknowledgement(), true).Acknowledgement()

			tc.malleate() // malleate mutates test data

			// retrieve module callbacks
			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			err = cbs.OnAcknowledgementPacket(s.chainA.GetContext(), packet, ack, relayerAddr)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}

			tc.expResult()
		})
	}
}

func (s *FeeTestSuite) TestOnTimeoutPacket() {
	var (
		packetID            channeltypes.PacketId
		packetFee           types.PacketFee
		refundAddr          sdk.AccAddress
		relayerAddr         sdk.AccAddress
		expRefundAccBalance sdk.Coins
		expPayeeAccBalance  sdk.Coins
	)

	testCases := []struct {
		name      string
		malleate  func()
		expPass   bool
		expResult func()
	}{
		{
			"success",
			func() {
				// retrieve the relayer acc balance and add the expected timeout fees
				relayerAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), relayerAddr, sdk.DefaultBondDenom))
				expPayeeAccBalance = relayerAccBalance.Add(packetFee.Fee.TimeoutFee...)

				// retrieve the refund acc balance and add the expected recv and ack fees
				refundAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom))
				expRefundAccBalance = refundAccBalance.Add(packetFee.Fee.RecvFee...).Add(packetFee.Fee.AckFee...)
			},
			true,
			func() {
				// assert that the packet fees have been distributed
				found := s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID)
				s.Require().False(found)

				relayerAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), relayerAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expPayeeAccBalance, sdk.NewCoins(relayerAccBalance))

				refundAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expRefundAccBalance, sdk.NewCoins(refundAccBalance))
			},
		},
		{
			"success: with registered payee address",
			func() {
				payeeAddr := s.chainA.SenderAccounts[2].SenderAccount.GetAddress()
				s.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
					s.chainA.GetContext(),
					s.chainA.SenderAccount.GetAddress().String(),
					payeeAddr.String(),
					s.path.EndpointA.ChannelID,
				)

				// retrieve the relayer acc balance and add the expected timeout fees
				payeeAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), payeeAddr, sdk.DefaultBondDenom))
				expPayeeAccBalance = payeeAccBalance.Add(packetFee.Fee.TimeoutFee...)

				// retrieve the refund acc balance and add the expected recv and ack fees
				refundAccBalance := sdk.NewCoins(s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom))
				expRefundAccBalance = refundAccBalance.Add(packetFee.Fee.RecvFee...).Add(packetFee.Fee.AckFee...)
			},
			true,
			func() {
				// assert that the packet fees have been distributed
				found := s.chainA.GetSimApp().IBCFeeKeeper.HasFeesInEscrow(s.chainA.GetContext(), packetID)
				s.Require().False(found)

				payeeAddr := s.chainA.SenderAccounts[2].SenderAccount.GetAddress()
				payeeAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), payeeAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expPayeeAccBalance, sdk.NewCoins(payeeAccBalance))

				refundAccBalance := s.chainA.GetSimApp().BankKeeper.GetBalance(s.chainA.GetContext(), refundAddr, sdk.DefaultBondDenom)
				s.Require().Equal(expRefundAccBalance, sdk.NewCoins(refundAccBalance))
			},
		},
		{
			"success: channel is not fee enabled",
			func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(s.chainA.GetContext(), s.path.EndpointA.ChannelConfig.PortID, s.path.EndpointA.ChannelID)
			},
			true,
			func() {},
		},
		{
			"success: fee module is disabled, skip fee logic",
			func() {
				lockFeeModule(s.chainA)
			},
			true,
			func() {
				s.Require().Equal(true, s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(s.chainA.GetContext()))
			},
		},
		{
			"success: no op if identified packet fee doesn't exist",
			func() {
				s.chainA.GetSimApp().IBCFeeKeeper.DeleteFeesInEscrow(s.chainA.GetContext(), packetID)
			},
			true,
			func() {},
		},
		{
			"success: fail to distribute timeout fee (blocked address), returned to refund account",
			func() {
				relayerAddr = s.chainA.GetSimApp().AccountKeeper.GetModuleAccount(s.chainA.GetContext(), transfertypes.ModuleName).GetAddress()
			},
			true,
			func() {},
		},
		{
			"fee distribution fails and fee module is locked when escrow account does not have sufficient funds",
			func() {
				err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(s.chainA.GetContext(), types.ModuleName, s.chainA.SenderAccount.GetAddress(), smallAmount)
				s.Require().NoError(err)
			},
			true,
			func() {
				s.Require().Equal(true, s.chainA.GetSimApp().IBCFeeKeeper.IsLocked(s.chainA.GetContext()))
			},
		},
		{
			"invalid registered payee address",
			func() {
				payeeAddr := "invalid-address"
				s.chainA.GetSimApp().IBCFeeKeeper.SetPayeeAddress(
					s.chainA.GetContext(),
					s.chainA.SenderAccount.GetAddress().String(),
					payeeAddr,
					s.path.EndpointA.ChannelID,
				)
			},
			false,
			func() {},
		},
		{
			"application callback fails",
			func() {
				s.chainA.GetSimApp().FeeMockModule.IBCApp.OnTimeoutPacket = func(_ sdk.Context, _ channeltypes.Packet, _ sdk.AccAddress) error {
					return fmt.Errorf("mock fee app callback fails")
				}
			},
			false,
			func() {},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.Setup(s.path)

			relayerAddr = s.chainA.SenderAccounts[0].SenderAccount.GetAddress()
			refundAddr = s.chainA.SenderAccounts[1].SenderAccount.GetAddress()

			packet := s.CreateMockPacket()
			packetID = channeltypes.NewPacketID(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
			packetFee = types.NewPacketFee(types.NewFee(defaultRecvFee, defaultAckFee, defaultTimeoutFee), refundAddr.String(), nil)

			s.chainA.GetSimApp().IBCFeeKeeper.SetFeesInEscrow(s.chainA.GetContext(), packetID, types.NewPacketFees([]types.PacketFee{packetFee}))
			err := s.chainA.GetSimApp().BankKeeper.SendCoinsFromAccountToModule(s.chainA.GetContext(), s.chainA.SenderAccount.GetAddress(), types.ModuleName, packetFee.Fee.Total())
			s.Require().NoError(err)

			tc.malleate() // malleate mutates test data

			// retrieve module callbacks
			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			err = cbs.OnTimeoutPacket(s.chainA.GetContext(), packet, relayerAddr)

			if tc.expPass {
				s.Require().NoError(err)
			} else {
				s.Require().Error(err)
			}

			tc.expResult()
		})
	}
}

func (s *FeeTestSuite) TestGetAppVersion() {
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
				path := ibctesting.NewPath(s.chainA, s.chainB)
				path.EndpointA.ChannelConfig.PortID = ibctesting.MockFeePort
				path.EndpointB.ChannelConfig.PortID = ibctesting.MockFeePort
				// by default a new path uses a non fee channel
				s.coordinator.Setup(path)
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
		s.Run(tc.name, func() {
			s.SetupTest()
			s.coordinator.Setup(s.path)

			portID = s.path.EndpointA.ChannelConfig.PortID
			channelID = s.path.EndpointA.ChannelID

			// malleate test case
			tc.malleate()

			module, _, err := s.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(s.chainA.GetContext(), ibctesting.MockFeePort)
			s.Require().NoError(err)

			cbs, ok := s.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			s.Require().True(ok)

			feeModule := cbs.(fee.IBCMiddleware)

			appVersion, found := feeModule.GetAppVersion(s.chainA.GetContext(), portID, channelID)

			if tc.expFound {
				s.Require().True(found)
				s.Require().Equal(expAppVersion, appVersion)
			} else {
				s.Require().False(found)
				s.Require().Empty(appVersion)
			}
		})
	}
}
