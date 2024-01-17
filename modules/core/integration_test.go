package ibc_test

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

// If packet receipts are pruned, it is possible to double spend
// by resubmitting the same proof used to process the original receive.
func (suite *IBCTestSuite) TestDoubleSpendAttackOnPrunedReceipts() {
	path := ibctesting.NewPath(suite.chainA, suite.chainB)

	// configure the initial path to create a regular transfer channel
	path.EndpointA.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointB.ChannelConfig.PortID = transfertypes.PortID
	path.EndpointA.ChannelConfig.Version = transfertypes.Version
	path.EndpointB.ChannelConfig.Version = transfertypes.Version

	suite.coordinator.Setup(path)

	// configure the channel upgrade to upgrade to an incentivized fee enabled transfer channel
	upgradeVersion := string(types.ModuleCdc.MustMarshalJSON(&types.Metadata{FeeVersion: types.Version, AppVersion: transfertypes.Version}))
	path.EndpointA.ChannelConfig.ProposedUpgrade.Fields.Version = upgradeVersion
	path.EndpointB.ChannelConfig.ProposedUpgrade.Fields.Version = upgradeVersion

	// setup double spend attack
	amount, ok := sdkmath.NewIntFromString("1000")
	suite.Require().True(ok)
	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)

	// send from chainA to chainB
	timeoutHeight := clienttypes.NewHeight(1, 110)
	msg := transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coin, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0, "")
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	suite.Require().NoError(err)

	// relay
	err = path.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// get proof of packet commitment on source
	packetKey := host.PacketCommitmentKey(packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	proof, proofHeight := path.EndpointA.Chain.QueryProof(packetKey)

	recvMsg := channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, path.EndpointB.Chain.SenderAccount.GetAddress().String())

	// receive on counterparty and update source client
	res, err = path.EndpointB.Chain.SendMsgs(recvMsg)
	suite.Require().NoError(err)

	// check that voucher exists on chain B
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))
	ibcCoin := sdk.NewCoin(voucherDenomTrace.IBCDenom(), coin.Amount)
	balance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	suite.Require().Equal(ibcCoin, balance)

	err = path.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	ack, err := ibctesting.ParseAckFromEvents(res.Events)
	suite.Require().NoError(err)

	err = path.EndpointA.AcknowledgePacket(packet, ack)
	suite.Require().NoError(err)

	err = path.EndpointA.ChanUpgradeInit()
	suite.Require().NoError(err)

	err = path.EndpointB.ChanUpgradeTry()
	suite.Require().NoError(err)

	err = path.EndpointA.ChanUpgradeAck()
	suite.Require().NoError(err)

	err = path.EndpointB.ChanUpgradeConfirm()
	suite.Require().NoError(err)

	err = path.EndpointA.ChanUpgradeOpen()
	suite.Require().NoError(err)

	// prune
	msgPrune := channeltypes.NewMsgPruneAcknowledgements(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, 1, path.EndpointB.Chain.SenderAccount.GetAddress().String())
	res, err = path.EndpointB.Chain.SendMsgs(msgPrune)
	suite.Require().NoError(err)

	recvMsg = channeltypes.NewMsgRecvPacket(packet, proof, proofHeight, path.EndpointB.Chain.SenderAccount.GetAddress().String())

	// double spend
	res, err = path.EndpointB.Chain.SendMsgs(recvMsg)
	suite.Require().NoError(err)

	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	suite.Require().Equal(ibcCoin, balance, "successfully double spent")
}
