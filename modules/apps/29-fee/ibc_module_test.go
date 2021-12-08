package fee_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	"github.com/cosmos/ibc-go/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/testing"
	"github.com/cosmos/ibc-go/testing/simapp"
)

var (
	validCoins   = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(100)}}
	validCoins2  = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(200)}}
	validCoins3  = sdk.Coins{sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(300)}}
	invalidCoins = sdk.Coins{sdk.Coin{Denom: "invalidDenom", Amount: sdk.NewInt(100)}}
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
		name          string
		malleate      func(suite *FeeTestSuite)
		sourceRelayer bool
	}{
		{
			"success",
			func(suite *FeeTestSuite) {
				suite.chainA.GetSimApp().IBCFeeKeeper.SetCounterpartyAddress(suite.chainA.GetContext(), suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String())
			},
			true,
		},
		{
			"source relayer is empty string",
			func(suite *FeeTestSuite) {},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.SetupConnections(suite.path)

			// open incentivized channel
			suite.coordinator.Setup(suite.path)

			// set up coins
			coin := ibctesting.TestCoin
			fungibleTokenPacket := transfertypes.NewFungibleTokenPacketData(
				coin.Denom,
				sdk.NewInt(100).Uint64(),
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
			)
			packet := channeltypes.NewPacket(
				fungibleTokenPacket.GetBytes(),
				suite.chainA.SenderAccount.GetSequence(),
				suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID,
				suite.path.EndpointB.ChannelConfig.PortID,
				suite.path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			// set up & escrow packet fees
			refundAcc := suite.chainA.SenderAccount.GetAddress()
			packetId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, 1)
			identifiedFee := types.NewIdentifiedPacketFee(
				packetId,
				types.Fee{
					ReceiveFee: validCoins,
					AckFee:     validCoins2,
					TimeoutFee: validCoins3,
				},
				refundAcc.String(),
				[]string{},
			)
			err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
			suite.Require().NoError(err)

			// set up module and callbacks
			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			// malleate test case
			tc.malleate(suite)

			result := cbs.OnRecvPacket(suite.chainA.GetContext(), packet, refundAcc)

			if tc.sourceRelayer {
				suite.Require().NoError(err, "unexpected error for case: %s", tc.name)
				ack := types.IncentivizedAcknowledgement{
					Result:                channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement(),
					ForwardRelayerAddress: suite.chainB.SenderAccount.GetAddress().String(),
				}
				suite.Require().Equal(result, ack)
			} else {
				suite.Require().NoError(err, "unexpected error for case: %s", tc.name)
				ack := types.IncentivizedAcknowledgement{
					Result:                channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement(),
					ForwardRelayerAddress: "",
				}
				suite.Require().Equal(result, ack)
			}
		})
	}
}

func (suite *FeeTestSuite) TestOnAcknowledgementPacket() {
	testCases := []struct {
		name     string
		malleate func(suite *FeeTestSuite)
		expPass  bool
	}{
		{
			"success",
			func(suite *FeeTestSuite) {
				packetId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, suite.chainA.SenderAccount.GetSequence())
				identifiedFee := types.NewIdentifiedPacketFee(
					packetId,
					types.Fee{
						ReceiveFee: validCoins,
						AckFee:     validCoins2,
						TimeoutFee: validCoins3,
					},
					suite.chainA.SenderAccount.GetAddress().String(),
					[]string{},
				)
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
				suite.Require().NoError(err)
			},
			true,
		},
		{
			"identified packet fee doesn't exist",
			func(suite *FeeTestSuite) {
				suite.Require().Error(types.ErrFeeNotFound)
			},
			false,
		},
		{
			"refundAccount doesn't exist",
			func(suite *FeeTestSuite) {
				packetId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, suite.chainA.SenderAccount.GetSequence())
				identifiedFee := types.NewIdentifiedPacketFee(
					packetId,
					types.Fee{
						ReceiveFee: validCoins,
						AckFee:     validCoins2,
						TimeoutFee: validCoins3,
					},
					suite.chainB.SenderAccount.GetAddress().String(),
					[]string{},
				)
				suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
				suite.Require().Error(types.ErrRefundAccNotFound)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.SetupConnections(suite.path)

			// open incentivized channel
			suite.coordinator.Setup(suite.path)

			// set up coins
			coin := ibctesting.TestCoin
			fungibleTokenPacket := transfertypes.NewFungibleTokenPacketData(
				coin.Denom,
				sdk.NewInt(100).Uint64(),
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
			)
			packet := channeltypes.NewPacket(
				fungibleTokenPacket.GetBytes(),
				suite.chainA.SenderAccount.GetSequence(),
				suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID,
				suite.path.EndpointB.ChannelConfig.PortID,
				suite.path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			// set up module and callbacks
			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			// malleate test case
			tc.malleate(suite)

			ack := types.IncentivizedAcknowledgement{
				Result:                channeltypes.NewResultAcknowledgement([]byte{1}).Acknowledgement(),
				ForwardRelayerAddress: suite.chainB.SenderAccount.GetAddress().String(),
			}
			ackBytes := ack.Acknowledgement()

			err = cbs.OnAcknowledgementPacket(suite.chainA.GetContext(), packet, ackBytes, suite.chainB.SenderAccount.GetAddress())
			if tc.expPass {
				suite.Require().NoError(err, "unexpected error for case: %s", tc.name)
			} else {
				suite.Require().Error(err, "%s expected error but returned none", tc.name)
			}
		})
	}
}

func (suite *FeeTestSuite) TestOnTimeoutPacket() {
	testCases := []struct {
		name     string
		malleate func(suite *FeeTestSuite)
		expPass  bool
	}{
		{
			"success",
			func(suite *FeeTestSuite) {
				packetId := channeltypes.NewPacketId(suite.path.EndpointA.ChannelID, suite.path.EndpointA.ChannelConfig.PortID, suite.chainA.SenderAccount.GetSequence())
				identifiedFee := types.NewIdentifiedPacketFee(
					packetId,
					types.Fee{
						ReceiveFee: validCoins,
						AckFee:     validCoins2,
						TimeoutFee: validCoins3,
					},
					suite.chainA.SenderAccount.GetAddress().String(),
					[]string{},
				)
				err := suite.chainA.GetSimApp().IBCFeeKeeper.EscrowPacketFee(suite.chainA.GetContext(), identifiedFee)
				suite.Require().NoError(err)

				// fund chain A's escrow path so that tokens can be unescrowed upon timeout
				escrow := transfertypes.GetEscrowAddress(suite.path.EndpointA.ChannelConfig.PortID, suite.path.EndpointA.ChannelID)
				trace := transfertypes.ParseDenomTrace(sdk.DefaultBondDenom)
				coin := sdk.NewCoin(trace.IBCDenom(), sdk.NewInt(100))

				suite.Require().NoError(simapp.FundAccount(suite.chainA.GetSimApp(), suite.chainA.GetContext(), escrow, sdk.NewCoins(coin)))
			},
			true,
		},
		{
			"identified packet fee doesn't exist",
			func(suite *FeeTestSuite) {},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()
			suite.coordinator.SetupConnections(suite.path)

			// open incentivized channel
			suite.coordinator.Setup(suite.path)

			// set up coins
			coin := ibctesting.TestCoin
			fungibleTokenPacket := transfertypes.NewFungibleTokenPacketData(
				coin.Denom,
				sdk.NewInt(100).Uint64(),
				suite.chainA.SenderAccount.GetAddress().String(),
				suite.chainB.SenderAccount.GetAddress().String(),
			)
			packet := channeltypes.NewPacket(
				fungibleTokenPacket.GetBytes(),
				suite.chainA.SenderAccount.GetSequence(),
				suite.path.EndpointA.ChannelConfig.PortID,
				suite.path.EndpointA.ChannelID,
				suite.path.EndpointB.ChannelConfig.PortID,
				suite.path.EndpointB.ChannelID,
				clienttypes.NewHeight(0, 100),
				0,
			)

			// set up module and callbacks
			module, _, err := suite.chainA.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainA.GetContext(), ibctesting.TransferPort)
			suite.Require().NoError(err)

			cbs, ok := suite.chainA.App.GetIBCKeeper().Router.GetRoute(module)
			suite.Require().True(ok)

			// malleate test case
			tc.malleate(suite)
			err = cbs.OnTimeoutPacket(suite.chainA.GetContext(), packet, suite.chainA.SenderAccount.GetAddress())

			if tc.expPass {
				suite.Require().NoError(err, "unexpected error for case: %s", tc.name)
			} else {
				suite.Require().Error(err, "%s expected error but returned none", tc.name)
			}
		})
	}
}
