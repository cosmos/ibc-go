package fee_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/cosmos/ibc-go/testing/simapp"
)

var (
	validCoins  = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	validCoins2 = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(200)}}
	validCoins3 = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(300)}}
)

// Tests OnChanOpenInit on ChainA
func (suite *FeeTestSuite) TestOnChanOpenInit() {
	testCases := []struct {
		name    string
		version string
		expPass bool
	}{
		{
			"valid fee middleware and transfer version",
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			true,
		},
		{
			"fee version not included, only perform transfer logic",
			transfertypes.Version,
			true,
		},
		{
			"invalid fee middleware version",
			channeltypes.MergeChannelVersions("otherfee28-1", transfertypes.Version),
			false,
		},
		{
			"invalid transfer version",
			channeltypes.MergeChannelVersions(types.Version, "wrongics20-1"),
			false,
		},
		{
			"incorrect wrapping delimiter",
			fmt.Sprintf("%s//%s", types.Version, transfertypes.Version),
			false,
		},
		{
			"transfer version not wrapped",
			types.Version,
			false,
		},
		{
			"hanging delimiter",
			fmt.Sprintf("%s:%s:", types.Version, transfertypes.Version),
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.name, func() {
			// reset suite
			suite.SetupTest()
			suite.coordinator.SetupConnections(suite.path)
			suite.path.EndpointA.ChannelID = ibctesting.FirstChannelID

			counterparty := channeltypes.NewCounterparty(suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)
			channel := &channeltypes.Channel{
				State:          channeltypes.INIT,
				Ordering:       channeltypes.UNORDERED,
				Counterparty:   counterparty,
				ConnectionHops: []string{suite.path.EndpointA.ConnectionID},
				Version:        tc.version,
			}

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			chanCap, err := suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, suite.path.EndpointA.ChannelID))
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
		version   string
		cpVersion string
		crossing  bool
		expPass   bool
	}{
		{
			"valid fee middleware and transfer version",
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			false,
			true,
		},
		{
			"valid transfer version on try and counterparty",
			transfertypes.Version,
			transfertypes.Version,
			false,
			true,
		},
		{
			"valid fee middleware and transfer version, crossing hellos",
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			true,
			true,
		},
		{
			"invalid fee middleware version",
			channeltypes.MergeChannelVersions("otherfee28-1", transfertypes.Version),
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			false,
			false,
		},
		{
			"invalid counterparty fee middleware version",
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			channeltypes.MergeChannelVersions("wrongfee29-1", transfertypes.Version),
			false,
			false,
		},
		{
			"invalid transfer version",
			channeltypes.MergeChannelVersions(types.Version, "wrongics20-1"),
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			false,
			false,
		},
		{
			"invalid counterparty transfer version",
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			channeltypes.MergeChannelVersions(types.Version, "wrongics20-1"),
			false,
			false,
		},
		{
			"transfer version not wrapped",
			types.Version,
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			false,
			false,
		},
		{
			"counterparty transfer version not wrapped",
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			types.Version,
			false,
			false,
		},
		{
			"fee version not included on try, but included in counterparty",
			transfertypes.Version,
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			false,
			false,
		},
		{
			"fee version not included",
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			transfertypes.Version,
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

			var (
				chanCap *capabilitytypes.Capability
				ok      bool
				err     error
			)
			if tc.crossing {
				suite.path.EndpointA.ChanOpenInit()
				chanCap, ok = suite.chainA.GetSimApp().ScopedTransferKeeper.GetCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, suite.path.EndpointA.ChannelID))
				suite.Require().True(ok)
			} else {
				chanCap, err = suite.chainA.App.GetScopedIBCKeeper().NewCapability(suite.chainA.GetContext(), host.ChannelCapabilityPath(ibctesting.TransferPort, suite.path.EndpointA.ChannelID))
				suite.Require().NoError(err)
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

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanOpenTry(suite.chainA.GetContext(), channel.Ordering, channel.GetConnectionHops(),
				suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, chanCap, counterparty, tc.version, tc.cpVersion)

			if tc.expPass {
				suite.Require().NoError(err, "unexpected error from version: %s", tc.version)
			} else {
				suite.Require().Error(err, "error not returned for version: %s", tc.version)
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
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			func(suite *FeeTestSuite) {},
			true,
		},
		{
			"invalid fee version",
			channeltypes.MergeChannelVersions("fee29-A", transfertypes.Version),
			func(suite *FeeTestSuite) {},
			false,
		},
		{
			"invalid transfer version",
			channeltypes.MergeChannelVersions(types.Version, "ics20-4"),
			func(suite *FeeTestSuite) {},
			false,
		},
		{
			"previous INIT set without fee, however counterparty set fee version", // note this can only happen with incompetent or malicious counterparty chain
			channeltypes.MergeChannelVersions(types.Version, transfertypes.Version),
			func(suite *FeeTestSuite) {
				// do the first steps without fee version, then pass the fee version as counterparty version in ChanOpenACK
				suite.path.EndpointA.ChannelConfig.Version = transfertypes.Version
				suite.path.EndpointB.ChannelConfig.Version = transfertypes.Version
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.SetupConnections(suite.path)

			// malleate test case
			tc.malleate(suite)

			suite.path.EndpointA.ChanOpenInit()
			suite.path.EndpointB.ChanOpenTry()

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			err = cbs.OnChanOpenAck(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID, tc.cpVersion)
			if tc.expPass {
				suite.Require().NoError(err, "unexpected error for case: %s", tc.name)
			} else {
				suite.Require().Error(err, "%s expected error but returned none", tc.name)
			}
		})
	}
}

// Tests OnChanCloseInit on chainA
func (suite *FeeTestSuite) TestOnChanCloseInit() {
	testCases := []struct {
		name     string
		setup    func(suite *FeeTestSuite)
		disabled bool
	}{
		{
			"success",
			func(suite *FeeTestSuite) {
				packetId := channeltypes.PacketId{
					PortId:    suite.path.EndpointA.ChannelConfig.PortID,
					ChannelId: suite.path.EndpointA.ChannelID,
					Sequence:  1,
				}
				refundAcc := suite.chainA.SenderAccount.GetAddress()
				identifiedFee := types.NewIdentifiedPacketFee(&packetId, types.Fee{validCoins, validCoins2, validCoins3}, refundAcc.String(), []string{})
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"module account balance insufficient",
			func(suite *FeeTestSuite) {
				packetId := channeltypes.PacketId{
					PortId:    suite.path.EndpointA.ChannelConfig.PortID,
					ChannelId: suite.path.EndpointA.ChannelID,
					Sequence:  1,
				}
				refundAcc := suite.chainA.SenderAccount.GetAddress()
				identifiedFee := types.NewIdentifiedPacketFee(&packetId, types.Fee{validCoins, validCoins2, validCoins3}, refundAcc.String(), []string{})
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
				suite.Require().NoError(err)

				suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, refundAcc, validCoins3)

				// set fee enabled on different channel
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), "portID7", "channel-7")
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path) // setup channel

			origBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress())

			tc.setup(suite)

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			if tc.disabled {
				suite.Require().True(
					suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID),

					"fee is not disabled on original channel: %s", suite.path.EndpointA.ChannelID,
				)
				suite.Require().True(
					suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), "portID7", "channel-7"),

					"fee is not disabled on other channel: %s", "channel-7",
				)
			} else {
				cbs.OnChanCloseInit(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
				afterBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress())
				suite.Require().Equal(origBal, afterBal, "balances of refund account not equal after all fees refunded")
			}
		})
	}
}

// Tests OnChanCloseConfirm on chainA
func (suite *FeeTestSuite) TestOnChanCloseConfirm() {
	testCases := []struct {
		name     string
		setup    func(suite *FeeTestSuite)
		disabled bool
	}{
		{
			"success",
			func(suite *FeeTestSuite) {
				packetId := channeltypes.PacketId{
					PortId:    suite.path.EndpointA.ChannelConfig.PortID,
					ChannelId: suite.path.EndpointA.ChannelID,
					Sequence:  1,
				}
				refundAcc := suite.chainA.SenderAccount.GetAddress()
				identifiedFee := types.NewIdentifiedPacketFee(&packetId, types.Fee{validCoins, validCoins2, validCoins3}, refundAcc.String(), []string{})
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
				suite.Require().NoError(err)
			},
			false,
		},
		{
			"module account balance insufficient",
			func(suite *FeeTestSuite) {
				packetId := channeltypes.PacketId{
					PortId:    suite.path.EndpointA.ChannelConfig.PortID,
					ChannelId: suite.path.EndpointA.ChannelID,
					Sequence:  1,
				}
				refundAcc := suite.chainA.SenderAccount.GetAddress()
				identifiedFee := types.NewIdentifiedPacketFee(&packetId, types.Fee{validCoins, validCoins2, validCoins3}, refundAcc.String(), []string{})
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
				suite.Require().NoError(err)

				suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, refundAcc, validCoins3)

				// set fee enabled on different channel
				suite.chainA.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainA.GetContext(), "portID7", "channel-7")
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path) // setup channel

			origBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress())

			tc.setup(suite)

			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			if tc.disabled {
				suite.Require().True(
					suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID),

					"fee is not disabled on original channel: %s", suite.path.EndpointA.ChannelID,
				)
				suite.Require().True(
					suite.chainA.GetSimApp().IBCFeeKeeper.IsFeeEnabled(suite.chainA.GetContext(), "portID7", "channel-7"),

					"fee is not disabled on other channel: %s", "channel-7",
				)
			} else {
				cbs.OnChanCloseConfirm(suite.chainA.GetContext(), suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
				afterBal := suite.chainA.GetSimApp().BankKeeper.GetAllBalances(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress())
				suite.Require().Equal(origBal, afterBal, "balances of refund account not equal after all fees refunded")
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
			"source relayer is empty string",
			func() {
				suite.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainB.GetContext(), suite.chainA.SenderAccount.GetAddress().String(), "")
			},
			false,
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
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.Setup(suite.path)

			// set up coin & ics20 packet
			coin := ibctesting.TestCoin

			// set up a different channel to make sure that the test will error if the destination channel of the packet is not fee enabled
			suite.path.EndpointB.ChannelID = "channel-1"
			suite.chainB.GetSimApp().IBCFeeKeeper.SetFeeEnabled(suite.chainB.GetContext(), suite.path.EndpointB.ChannelConfig.PortID, suite.path.EndpointB.ChannelID)
			suite.chainB.GetSimApp().IBCFeeKeeper.DeleteFeeEnabled(suite.chainB.GetContext(), suite.path.EndpointB.ChannelConfig.PortID, "channel-0")

			packet := suite.CreateICS20Packet(coin)

			// set up module and callbacks
			module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainB.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			suite.chainB.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainB.GetContext(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String())

			// malleate test case
			tc.malleate()

			result := cbs.OnRecvPacket(suite.chainB.GetContext(), packet, suite.chainA.SenderAccount.GetAddress())

			switch {
			case !tc.feeEnabled:
				ack := channeltypes.NewResultAcknowledgement([]byte{1})
				suite.Require().Equal(ack, result)

			case tc.forwardRelayer:
				ack := types.IncentivizedAcknowledgement{
					Result:                channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement(),
					ForwardRelayerAddress: suite.chainB.SenderAccount.GetAddress().String(),
				}
				suite.Require().Equal(ack, result)

			case !tc.forwardRelayer:
				ack := types.IncentivizedAcknowledgement{
					Result:                channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement(),
					ForwardRelayerAddress: "",
				}
				suite.Require().Equal(ack, result)
			}
		})
	}
}

// different channel than sending chain
func (suite *FeeTestSuite) TestOnAcknowledgementPacket() {
	var (
		ack                    []byte
		identifiedFee          *types.IdentifiedPacketFee
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
				expectedRelayerBalance = identifiedFee.Fee.ReceiveFee.Add(identifiedFee.Fee.AckFee[0])
			},
			true,
		},
		{
			"no op success without a packet fee",
			func() {
				packetId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, suite.chainA.SenderAccount.GetSequence())
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeInEscrow(suite.chainA.GetContext(), packetId)

				ack = types.IncentivizedAcknowledgement{
					Result:                channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement(),
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
				ack = channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement()

				expectedBalance = originalBalance
			},
			true,
		},
		{
			"fail on distribute receive fee (blocked address)",
			func() {
				blockedAddr := suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), types.ModuleName).GetAddress()

				ack = types.IncentivizedAcknowledgement{
					Result:                channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement(),
					ForwardRelayerAddress: blockedAddr.String(),
				}.Acknowledgement()

				expectedBalance = originalBalance.Add(identifiedFee.Fee.AckFee[0])
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			expectedRelayerBalance = sdk.Coins{} // reset

			// open incentivized channel
			suite.coordinator.Setup(suite.path)

			// set up coin & ics20 packet
			coin := ibctesting.TestCoin
			packet := suite.CreateICS20Packet(coin)

			// set up module and callbacks
			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			// escrow the packet fee
			packetId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, suite.chainA.SenderAccount.GetSequence())
			identifiedFee = types.NewIdentifiedPacketFee(
				packetId,
				types.Fee{
					ReceiveFee: validCoins,
					AckFee:     validCoins2,
					TimeoutFee: validCoins3,
				},
				suite.chainA.SenderAccount.GetAddress().String(),
				[]string{},
			)
			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
			suite.Require().NoError(err)

			relayerAddr := suite.chainB.SenderAccount.GetAddress()

			// must be changed explicitly
			ack = types.IncentivizedAcknowledgement{
				Result:                channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement(),
				ForwardRelayerAddress: relayerAddr.String(),
			}.Acknowledgement()

			// log original sender balance
			// NOTE: balance is logged after escrowing tokens
			originalBalance = sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))

			// default to success case
			expectedBalance = originalBalance.
				Add(identifiedFee.Fee.TimeoutFee[0])

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
		identifiedFee   *types.IdentifiedPacketFee
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

				expectedBalance = originalBalance.Add(ibctesting.TestCoin) // timeout refund for ics20 transfer
			},
			false,
		},
		{
			"no op if identified packet fee doesn't exist",
			func() {
				// delete packet fee
				packetId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, suite.chainA.SenderAccount.GetSequence())
				suite.chainA.GetSimApp().IBCFeeKeeper.DeleteFeeInEscrow(suite.chainA.GetContext(), packetId)

				expectedBalance = originalBalance.Add(ibctesting.TestCoin) // timeout refund for ics20 transfer
			},
			false,
		},
		{
			"distribute fee fails for timeout fee (blocked address)",
			func() {
				relayerAddr = suite.chainA.GetSimApp().AccountKeeper.GetModuleAccount(suite.chainA.GetContext(), types.ModuleName).GetAddress()

				expectedBalance = originalBalance.
					Add(identifiedFee.Fee.ReceiveFee[0]).
					Add(identifiedFee.Fee.AckFee[0]).
					Add(ibctesting.TestCoin) // timeout refund for ics20 transfer
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// open incentivized channel
			suite.coordinator.Setup(suite.path)

			// set up coin & create ics20 packet
			coin := ibctesting.TestCoin
			packet := suite.CreateICS20Packet(coin)

			// setup for ics20: fund chain A's escrow path so that tokens can be unescrowed upon timeout
			escrow := transfertypes.GetEscrowAddress(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
			suite.Require().NoError(simapp.FundAccount(suite.chainA.GetSimApp(), suite.chainA.GetContext(), escrow, sdk.NewCoins(coin)))

			// set up module and callbacks
			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			packetId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, suite.chainA.SenderAccount.GetSequence())

			// must be explicitly changed
			relayerAddr = suite.chainB.SenderAccount.GetAddress()

			identifiedFee = types.NewIdentifiedPacketFee(
				packetId,
				types.Fee{
					ReceiveFee: validCoins,
					AckFee:     validCoins2,
					TimeoutFee: validCoins3,
				},
				suite.chainA.SenderAccount.GetAddress().String(),
				[]string{},
			)

			err = suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
			suite.Require().NoError(err)

			// log original sender balance
			// NOTE: balance is logged after escrowing tokens
			originalBalance = sdk.NewCoins(suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress(), ibctesting.TestCoin.Denom))

			// default to success case
			expectedBalance = originalBalance.
				Add(identifiedFee.Fee.ReceiveFee[0]).
				Add(identifiedFee.Fee.AckFee[0]).
				Add(coin) // timeout refund from ics20 transfer

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
				suite.Require().Equal(
					identifiedFee.Fee.TimeoutFee,
					relayerBalance,
				)
			} else {
				suite.Require().Empty(relayerBalance)
			}
		})
	}
}
