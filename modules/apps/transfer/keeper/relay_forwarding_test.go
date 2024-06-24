package keeper_test

import (
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
)

func (suite *KeeperTestSuite) TestPathForwarding() {
	amount := sdkmath.NewInt(100)

	// setup
	// 2 transfer channels between chain A and chain B
	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[1].SenderAccount
	forwarding := types.NewForwarding(false, types.Hop{
		PortId:    path2.EndpointA.ChannelConfig.PortID,
		ChannelId: path2.EndpointA.ChannelID,
	})

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(), "",
		forwarding,
	)
	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, packet.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packet, forwardedPacket)
}

func (suite *KeeperTestSuite) TestEscrowsAreSetAfterForwarding() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topolgy:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake
		We want to trigger:
		1. A sends B over channel0.
		2. B onRecv . 2.1(B sends A over channel1) Atomic Actions
		At this point we want to assert:
		A: escrowA = amount,denom
		B: escrowB = amount,transfer/channel-0/denom
	*/

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()
	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[1].SenderAccount
	forwarding := types.NewForwarding(false, types.Hop{
		PortId:    path2.EndpointB.ChannelConfig.PortID,
		ChannelId: path2.EndpointB.ChannelID,
	})

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(), "",
		forwarding,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainA.Amount)

	// denom path: transfer/channel-0
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that Escrow B has amount
	coin = sdk.NewCoin(denom.IBCDenom(), amount)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)
}

// This test is probably overcomplicated. Could have used RecvPacketWithResult directly.
func (suite *KeeperTestSuite) TestHappyPathForwarding() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topolgy:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain A
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake
		We want to trigger:
		1. A sends B over channel0.
		2. B onRecv . 2.1(B sends A over channel1) Atomic Actions
		At this point we want to assert:
		A: escrowA = amount,denom
		B: escrowB = amount,transfer/channel-0/denom
		3. A OnRecv
		At this point we want to assert:
		C: finalReceiver = amount,transfer/channel-1/transfer/channel-0/denom
	*/

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path2.Setup()

	// transfer/channel-1/transfer/channel-0/denom
	denomABA := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID), types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that initially the final receiver address has 0 ABA coins
	coin := sdk.NewCoin(denomABA.IBCDenom(), amount)
	preCoinOnA := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccounts[1].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), preCoinOnA.Amount, "final receiver has not zero balance")

	coin = sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[1].SenderAccount
	forwarding := types.NewForwarding(false, types.Hop{
		PortId:    path2.EndpointB.ChannelConfig.PortID,
		ChannelId: path2.EndpointB.ChannelID,
	})

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(), "",
		forwarding,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	forwardingPacketData := types.NewForwardingPacketData("", forwarding.Hops...)
	denom := types.Denom{Base: sdk.DefaultBondDenom}
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, sender.GetAddress().String(), receiver.GetAddress().String(), "", forwardingPacketData)
	packetRecv := channeltypes.NewPacket(data.GetBytes(), 2, path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, clienttypes.ZeroHeight(), suite.chainA.GetTimeoutTimestamp())

	err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packetRecv, data)
	// If forwarding has been triggered then the async must be true.
	suite.Require().Nil(err)

	// denomTrace path: transfer/channel-0
	denom = types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that Escrow B has amount
	coin = sdk.NewCoin(denom.IBCDenom(), amount)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(amount, totalEscrowChainB.Amount, "escrow account on B is different than amount")

	// Check that Escrow A has amount
	coin = sdk.NewCoin(sdk.DefaultBondDenom, amount)
	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(amount, totalEscrowChainA.Amount, "escrow account on A is different than amount")

	// Now during the onRecvPacket above a new msgTransfer has been sent
	// We need to receive the packet on the final hand

	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	data = types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, suite.chainA.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String(), receiver.GetAddress().String(), "", types.ForwardingPacketData{})
	packetRecv = channeltypes.NewPacket(data.GetBytes(), 3, path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID, path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)

	// execute onRecvPacket, when chaninA receives the tokens the escrow amount on B should increase to amount
	err = suite.chainA.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainA.GetContext(), packetRecv, data)
	suite.Require().NoError(err)

	// Check that the final receiver has received the expected tokens.
	coin = sdk.NewCoin(denomABA.IBCDenom(), amount)
	postCoinOnA := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccounts[1].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnA.Amount, "final receiver balance has not increased")
}

// Simplification of the above test.
func (suite *KeeperTestSuite) TestSimplifiedHappyPathForwarding() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topolgy:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain C
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake
		We want to trigger:
		1. Single transfer forwarding token from A -> B -> C
		2. B onRecv . 2.1(B sends C over channel1) Atomic Actions
		At this point we want to assert:
		A: escrowA = amount,denom
		B: escrowB = amount,transfer/channel-0/denom
	*/

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	path2.Setup()
	coinOnA := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainC.SenderAccounts[0].SenderAccount
	forwarding := types.NewForwarding(false, types.Hop{
		PortId:    path2.EndpointA.ChannelConfig.PortID,
		ChannelId: path2.EndpointA.ChannelID,
	})

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coinOnA),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(), "",
		forwarding,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packetFromAtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromAtoB)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointB.RecvPacketWithResult(packetFromAtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow A has amount
	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coinOnA.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainA.Amount)

	// denomTrace path: transfer/channel-0
	denomTrace := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that Escrow B has amount
	coinOnB := sdk.NewCoin(denomTrace.IBCDenom(), amount)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coinOnB.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoC)

	err = path2.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// B should now have deleted the forwarded packet.
	_, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromAtoB.DestinationPort, packetFromAtoB.DestinationChannel, packetFromAtoB.Sequence)
	suite.Require().False(found, "Chain B should have deleted its forwarded packet")

	result, err = path2.EndpointB.RecvPacketWithResult(packetFromBtoC)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// transfer/channel-1/transfer/channel-0/denom
	denomTraceABC := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID), types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that the final receiver has received the expected tokens.
	coinOnC := sdk.NewCoin(denomTraceABC.IBCDenom(), amount)
	balanceOnC := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), receiver.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), balanceOnC.Amount, "final receiver balance has not increased")

	successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	successAckBz := channeltypes.CommitAcknowledgement(successAck.Acknowledgement())
	ackOnC := suite.chainC.GetAcknowledgement(packetFromBtoC)
	suite.Require().Equal(successAckBz, ackOnC)

	// Ack back to B
	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = path2.EndpointA.AcknowledgePacket(packetFromBtoC, successAck.Acknowledgement())
	suite.Require().NoError(err)

	ackOnB := suite.chainB.GetAcknowledgement(packetFromAtoB)
	suite.Require().Equal(successAckBz, ackOnB)

	// Ack back to A
	err = path1.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = path1.EndpointA.AcknowledgePacket(packetFromAtoB, successAck.Acknowledgement())
	suite.Require().NoError(err)
}

// TestAcknowledgementFailureWithMiddleChainAsNativeTokenSource tests a failure in the last hop where the
// middle chain is native source when receiving and sending the packet. In other words, the middle chain's native
// token has been sent to chain C, and the multi-hop transfer from C -> B -> A has chain B being the source of
// the token both when receiving and forwarding (sending).
func (suite *KeeperTestSuite) TestAcknowledgementFailureWithMiddleChainAsNativeTokenSource() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topology:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-0) chain C
		stake                  transfer/channel-0/stake           transfer/channel-0/transfer/channel-0/stake
		We want to trigger:
			1. Transfer from B to C
			2. Single transfer forwarding token from C -> B -> A
				2.1 The ack fails on the last hop (chain A)
				2.2 Propagate the error back to C
			3. Verify all the balances are updated as expected
	*/

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	path2.Setup()

	coinOnB := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	setupSender := suite.chainB.SenderAccounts[0].SenderAccount
	setupReceiver := suite.chainC.SenderAccounts[0].SenderAccount

	setupTransferMsg := types.NewMsgTransfer(
		path2.EndpointA.ChannelConfig.PortID,
		path2.EndpointA.ChannelID,
		sdk.NewCoins(coinOnB),
		setupSender.GetAddress().String(),
		setupReceiver.GetAddress().String(),
		suite.chainB.GetTimeoutHeight(),
		0, "",
		types.Forwarding{},
	)

	result, err := suite.chainB.SendMsgs(setupTransferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainC
	packetFromBToC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBToC)

	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointB.RecvPacketWithResult(packetFromBToC)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that EscrowBtoC has amount
	escrowAddressBtoC := types.GetEscrowAddress(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID)
	escrowBalancBtoC := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoC, coinOnB.GetDenom())
	suite.Require().Equal(amount, escrowBalancBtoC.Amount)

	// Check that receiver has the expected tokens
	denomOnC := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID))
	coinOnC := sdk.NewCoin(denomOnC.IBCDenom(), amount)
	balanceOnC := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), setupReceiver.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(amount, balanceOnC.Amount)

	// Now we start the transfer from C -> B -> A
	sender := suite.chainC.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[0].SenderAccount

	forwarding := types.NewForwarding(false, types.Hop{
		PortId:    path1.EndpointB.ChannelConfig.PortID,
		ChannelId: path1.EndpointB.ChannelID,
	})

	forwardTransfer := types.NewMsgTransfer(
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		sdk.NewCoins(coinOnC),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(),
		"",
		forwarding,
	)

	result, err = suite.chainC.SendMsgs(forwardTransfer)
	suite.Require().NoError(err) // message committed

	// Check that Escrow C has unescrowed the amount
	totalEscrowChainC := suite.chainC.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainC.GetContext(), coinOnC.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), totalEscrowChainC.Amount)

	// parse the packet from result events and recv packet on chainB
	packetFromCtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromCtoB)

	err = path2.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointA.RecvPacketWithResult(packetFromCtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that escrow has been moved from EscrowBtoC to EscrowBtoA
	escrowBalancBtoC = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoC, coinOnB.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), escrowBalancBtoC.Amount)

	escrowAddressBtoA := types.GetEscrowAddress(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID)
	escrowBalanceBtoA := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoA, coinOnB.GetDenom())
	suite.Require().Equal(amount, escrowBalanceBtoA.Amount)

	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, packetFromCtoB.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packetFromCtoB, forwardedPacket)

	// Now we can receive the packet on A where we want to trigger an error
	packetFromBtoA, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoA)

	// turn off receive on chain A to trigger an error
	suite.chainA.GetSimApp().TransferKeeper.SetParams(suite.chainA.GetContext(), types.Params{
		SendEnabled:    true,
		ReceiveEnabled: false,
	})

	err = path1.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointA.RecvPacketWithResult(packetFromBtoA)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// An error ack is now written on chainA
	// Now we need to propagate the error to B and C
	errorAckOnA := channeltypes.NewErrorAcknowledgement(types.ErrReceiveDisabled)
	errorAckCommitmentOnA := channeltypes.CommitAcknowledgement(errorAckOnA.Acknowledgement())
	ackOnA := suite.chainA.GetAcknowledgement(packetFromBtoA)
	suite.Require().Equal(errorAckCommitmentOnA, ackOnA)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = path1.EndpointB.AcknowledgePacket(packetFromBtoA, errorAckOnA.Acknowledgement())
	suite.Require().NoError(err)

	errorAckOnB := channeltypes.NewErrorAcknowledgement(types.ErrForwardedPacketFailed)
	errorAckCommitmentOnB := channeltypes.CommitAcknowledgement(errorAckOnB.Acknowledgement())
	ackOnB := suite.chainB.GetAcknowledgement(packetFromCtoB)
	suite.Require().Equal(errorAckCommitmentOnB, ackOnB)

	// Check that escrow has been moved back from EscrowBtoA to EscrowBtoC
	escrowBalanceBtoA = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoA, coinOnB.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), escrowBalanceBtoA.Amount)

	escrowBalancBtoC = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoC, coinOnB.GetDenom())
	suite.Require().Equal(amount, escrowBalancBtoC.Amount)

	// Check the status of account on chain C before executing ack.
	balanceOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), setupReceiver.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), balanceOnC.Amount)

	// Propagate the error to C
	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = path2.EndpointB.AcknowledgePacket(packetFromCtoB, errorAckOnB.Acknowledgement())
	suite.Require().NoError(err)

	// Check that everything has been reverted
	//
	// Check the vouchers have been refunded on C
	balanceOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), setupReceiver.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(amount, balanceOnC.Amount, "final receiver balance has not increased")
}

// TestAcknowledgementFailureWithMiddleChainAsNotBeingTokenSource tests a failure in the last hop where the middle chain
// is not source of the token when receiving or sending the packet. In other words, the middle chain's is sent
// (and forwarding) someone else's native token (in this case chain C).
func (suite *KeeperTestSuite) TestAcknowledgementFailureWithMiddleChainAsNotBeingTokenSource() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topology:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-0) chain C
		stake                  transfer/channel-0/stake           transfer/channel-0/transfer/channel-0/stake
		We want to trigger:
			1. Single transfer forwarding token from C -> B -> A
			1.1 The ack fails on the last hop
			1.2 Propagate the error back to C
			2. Verify all the balances are updated as expected
	*/

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	path2.Setup()

	// Now we start the transfer from C -> B -> A
	coinOnC := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainC.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[0].SenderAccount

	forwarding := types.NewForwarding(false, types.Hop{
		PortId:    path1.EndpointB.ChannelConfig.PortID,
		ChannelId: path1.EndpointB.ChannelID,
	})

	forwardTransfer := types.NewMsgTransfer(
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		sdk.NewCoins(coinOnC),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(),
		"",
		forwarding,
	)

	balanceOnCBefore := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), sender.GetAddress(), coinOnC.GetDenom())

	result, err := suite.chainC.SendMsgs(forwardTransfer)
	suite.Require().NoError(err) // message committed

	// Check that Escrow C has amount
	totalEscrowChainC := suite.chainC.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainC.GetContext(), coinOnC.GetDenom())
	suite.Require().Equal(amount, totalEscrowChainC.Amount)

	packetFromCtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromCtoB)

	err = path2.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointA.RecvPacketWithResult(packetFromCtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow B has amount
	denomOnB := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID))
	coinOnB := sdk.NewCoin(denomOnB.IBCDenom(), amount)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coinOnB.GetDenom())
	suite.Require().Equal(amount, totalEscrowChainB.Amount)

	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, packetFromCtoB.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packetFromCtoB, forwardedPacket)

	// Now we can receive the packet on A where we want to trigger an error
	packetFromBtoA, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoA)

	// turn off receive on chain A to trigger an error
	suite.chainA.GetSimApp().TransferKeeper.SetParams(suite.chainA.GetContext(), types.Params{
		SendEnabled:    true,
		ReceiveEnabled: false,
	})

	err = path1.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointA.RecvPacketWithResult(packetFromBtoA)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// An error ack is now written on chainA
	// Now we need to propagate the error to B and C
	errorAckOnA := channeltypes.NewErrorAcknowledgement(types.ErrReceiveDisabled)
	errorAckCommitmentOnA := channeltypes.CommitAcknowledgement(errorAckOnA.Acknowledgement())
	ackOnA := suite.chainA.GetAcknowledgement(packetFromBtoA)
	suite.Require().Equal(errorAckCommitmentOnA, ackOnA)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = path1.EndpointB.AcknowledgePacket(packetFromBtoA, errorAckOnA.Acknowledgement())
	suite.Require().NoError(err)

	errorAckOnB := channeltypes.NewErrorAcknowledgement(types.ErrForwardedPacketFailed)
	errorAckCommitmentOnB := channeltypes.CommitAcknowledgement(errorAckOnB.Acknowledgement())
	ackOnB := suite.chainB.GetAcknowledgement(packetFromCtoB)
	suite.Require().Equal(errorAckCommitmentOnB, ackOnB)

	// Check that escrow has been burnt on B
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coinOnB.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), totalEscrowChainB.Amount)

	// Check the status of account on chain C before executing ack.
	balanceOnC := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), sender.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(balanceOnCBefore.SubAmount(amount).Amount, balanceOnC.Amount)

	// Propagate the error to C
	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = path2.EndpointB.AcknowledgePacket(packetFromCtoB, errorAckOnB.Acknowledgement())
	suite.Require().NoError(err)

	// Check that everything has been reverted
	//
	// Check the token has been returned to the sender on C
	totalEscrowChainC = suite.chainC.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainC.GetContext(), coinOnC.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), totalEscrowChainC.Amount)

	balanceOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), sender.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(balanceOnCBefore.Amount, balanceOnC.Amount, "final receiver balance has not increased")
}

// This tests a failure in the last hop where the middle chain as IBC denom source when receiving and sending the packet.
// In other words, an IBC denom from the middle chain's sent to chain C, and the multi-hop
// transfer from C -> B -> A has chain B being the source of the token both when receiving and forwarding (sending).
// Previously referenced as Acknowledgement Failure Scenario 5
func (suite *KeeperTestSuite) TestAcknowledgementFailureWithMiddleChainAsIBCTokenSource() {
	amount := sdkmath.NewInt(100)
	/*
		Given the following topolgy:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-1) chain C
		stake                  transfer/channel-0/stake           transfer/channel-1/transfer/channel-0/stake
		We want to trigger:
		0. A sends B over channel0 [path1]
		1. B sends C over channel1 [path2].
		2. C recvs - This represent the checkpoint we will need to verify at the of the test
		3. C --> [path2] B --> [path1] A.
		4. OnRecv in B works properly and trigger the packet forwarding to A
		5. Modify the balance of escrowA to cause an error during the onRecv
		6. OnRecv on A fails. Error Ack is written in A, relayed to B and finally to C.
		At this point we want to assert:
		Everything has been reverted at checkpoint values.
		- C has amount of transfer/channel-1/transfer/channel-0/stake
		- B totalEscrow has amount of transfer/channel-0/stake
	*/

	// Testing Topology

	path1 := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	path1.Setup()

	path2 := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	path2.Setup()

	// First we want to execute 0.

	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainB.SenderAccounts[0].SenderAccount

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		types.Forwarding{},
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packetFromAtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromAtoB)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointB.RecvPacketWithResult(packetFromAtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow B has amount
	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainA.Amount)

	// transfer/channel-0/denom
	denomAB := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check the coins have been received on B
	coin = sdk.NewCoin(denomAB.IBCDenom(), amount)
	postCoinOnB := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnB.Amount, "final receiver balance has not increased")

	// A --> B Simple transfer happened properly.

	// Now we want to trigger B -> C
	sender = suite.chainB.SenderAccounts[0].SenderAccount
	receiver = suite.chainC.SenderAccounts[0].SenderAccount

	transferMsg = types.NewMsgTransfer(
		path2.EndpointA.ChannelConfig.PortID,
		path2.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		types.Forwarding{},
	)

	result, err = suite.chainB.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoC)

	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointB.RecvPacketWithResult(packetFromBtoC)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow B has amount
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	// transfer/channel-1/transfer/channel-0/denom
	denomABC := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID), types.NewTrace(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID))

	// Check the coins have been received on C
	coin = sdk.NewCoin(denomABC.IBCDenom(), amount)
	postCoinOnC := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnC.Amount, "final receiver balance has not increased")

	// B -> C Simple transfer happened properly.

	// Now we want to trigger C -> B -> A
	// The coin we want to send out is exactly the one we received on C
	coin = sdk.NewCoin(denomABC.IBCDenom(), amount)

	sender = suite.chainC.SenderAccounts[0].SenderAccount
	receiver = suite.chainA.SenderAccounts[0].SenderAccount // Receiver is the A chain account

	forwarding := types.NewForwarding(false, types.Hop{
		PortId:    path1.EndpointB.ChannelConfig.PortID,
		ChannelId: path1.EndpointB.ChannelID,
	})

	transferMsg = types.NewMsgTransfer(
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(), "",
		forwarding,
	)

	result, err = suite.chainC.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// Voucher have been burned on chain C
	postCoinOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnC.Amount, "Vouchers have not been burned")

	// parse the packet from result events and recv packet on chainB
	packetFromCtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromCtoB)

	err = path2.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointA.RecvPacketWithResult(packetFromCtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// We have successfully received the packet on B and forwarded it to A.
	// Lets try to retrieve it in order to save it
	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, packetFromCtoB.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packetFromCtoB, forwardedPacket)

	// Voucher have been burned on chain B
	coin = sdk.NewCoin(denomAB.IBCDenom(), amount)
	postCoinOnB = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnB.Amount, "Vouchers have not been burned")

	// Now we can receive the packet on A.
	// To trigger an error during the OnRecv, we have to manipulate the balance present in the escrow of A
	// of denom

	// parse the packet from result events and recv packet on chainA
	packetFromBtoA, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoA)

	// turn off receive on chain A to trigger an error
	suite.chainA.GetSimApp().TransferKeeper.SetParams(suite.chainA.GetContext(), types.Params{
		SendEnabled:    true,
		ReceiveEnabled: false,
	})

	err = path1.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path1.EndpointA.RecvPacketWithResult(packetFromBtoA)
	suite.Require().NoError(err)

	// An error ack has been written on chainA
	// Now we need to propagate it back to chainB and chainC
	packetSequenceOnA, err := ibctesting.ParsePacketSequenceFromEvents(result.Events)
	suite.Require().NoError(err)

	errorAckOnA := channeltypes.NewErrorAcknowledgement(types.ErrReceiveDisabled)
	errorAckCommitmentOnA := channeltypes.CommitAcknowledgement(errorAckOnA.Acknowledgement())
	ackOnC, found := suite.chainA.GetSimApp().GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainA.GetContext(), path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, packetSequenceOnA)
	suite.Require().True(found)
	suite.Require().Equal(errorAckCommitmentOnA, ackOnC)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = path1.EndpointB.AcknowledgePacket(packetFromBtoA, errorAckOnA.Acknowledgement())
	suite.Require().NoError(err)

	// Check that B deleted the forwarded packet.
	_, found = suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), forwardedPacket.SourcePort, forwardedPacket.SourceChannel, forwardedPacket.Sequence)
	suite.Require().False(found, "chain B should have deleted the forwarded packet mapping")

	// Check that Escrow B has been refunded amount
	coin = sdk.NewCoin(denomAB.IBCDenom(), amount)
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	errorAckOnB := channeltypes.NewErrorAcknowledgement(types.ErrForwardedPacketFailed)
	errorAckCommitmentOnB := channeltypes.CommitAcknowledgement(errorAckOnB.Acknowledgement())
	ackOnB := suite.chainB.GetAcknowledgement(packetFromCtoB)
	suite.Require().Equal(errorAckCommitmentOnB, ackOnB)

	// Check the status of account on chain C before executing ack.
	coin = sdk.NewCoin(denomABC.IBCDenom(), amount)
	postCoinOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnC.Amount, "Final Hop balance has been refunded before Ack execution")

	// Execute ack
	err = path2.EndpointB.AcknowledgePacket(packetFromCtoB, errorAckOnB.Acknowledgement())
	suite.Require().NoError(err)

	// Check that everything has been reverted
	//
	// Check the vouchers transfer/channel-1/transfer/channel-0/denom have been refunded on C
	coin = sdk.NewCoin(denomABC.IBCDenom(), amount)
	postCoinOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnC.Amount, "final receiver balance has not increased")

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)
}

/*
// TODO
	Test scenarios for failures ack
Check out the notion page: https://www.notion.so/interchain/ICS20-v2-path-forwarding-091f1ac788e84a538261c5a247cb5924
// TODO
Test async ack is properly relayed to middle hop after forwarding transfer completition
// TODO
Tiemout during forwarding after middle hop execution reverts properly the state changes
*/

func (suite *KeeperTestSuite) setupForwardingPaths() (pathAtoB, pathBtoC *ibctesting.Path) {
	pathAtoB = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	pathBtoC = ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	pathAtoB.Setup()
	pathBtoC.Setup()
	return pathAtoB, pathBtoC
}

type amountType int

const (
	escrow amountType = iota
	balance
)

func (suite *KeeperTestSuite) assertAmountOnChain(chain *ibctesting.TestChain, balanceType amountType, amount sdkmath.Int, denom string) {
	var total sdk.Coin
	switch balanceType {
	case escrow:
		total = chain.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(chain.GetContext(), denom)
	case balance:
		total = chain.GetSimApp().BankKeeper.GetBalance(chain.GetContext(), chain.SenderAccounts[0].SenderAccount.GetAddress(), denom)
	default:
		suite.Fail("invalid amountType %s", balanceType)
	}
	suite.Require().Equal(amount, total.Amount, fmt.Sprintf("Chain %s: got balance of %d, wanted %d", chain.Name(), total.Amount, amount))
}

// TestOnTimeoutPacketForwarding tests the scenario in which a packet goes from
// A to C, using B as a forwarding hop. The packet times out when going to C
// from B and we verify that funds are properly returned to A.
func (suite *KeeperTestSuite) TestOnTimeoutPacketForwarding() {
	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	amount := sdkmath.NewInt(100)
	coin := sdk.NewCoin(sdk.DefaultBondDenom, amount)
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainC.SenderAccounts[0].SenderAccount

	denomA := types.NewDenom(coin.Denom)
	denomAB := types.NewDenom(coin.Denom, types.NewTrace(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	denomABC := types.NewDenom(coin.Denom, append(denomAB.Trace, types.NewTrace(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID))...)

	originalABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), sender.GetAddress(), coin.Denom)

	forwarding := types.Forwarding{
		Hops: []types.Hop{
			{
				PortId:    pathBtoC.EndpointA.ChannelConfig.PortID,
				ChannelId: pathBtoC.EndpointA.ChannelID,
			},
		},
	}

	transferMsg := types.NewMsgTransfer(
		pathAtoB.EndpointA.ChannelConfig.PortID,
		pathAtoB.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		uint64(suite.chainA.GetContext().BlockTime().Add(time.Minute*5).UnixNano()),
		"",
		forwarding,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// Receive packet on B.
	result, err = pathAtoB.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// Make sure funds went from A to B's escrow account.
	suite.assertAmountOnChain(suite.chainA, balance, originalABalance.Amount.Sub(amount), denomA.IBCDenom())
	suite.assertAmountOnChain(suite.chainB, escrow, amount, denomAB.IBCDenom())

	// Check that forwarded packet exists
	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID, packet.Sequence)
	suite.Require().True(found, "Chain B has no forwarded packet")
	suite.Require().Equal(packet, forwardedPacket, "ForwardedPacket stored in ChainB is not the same that was sent")

	address := suite.chainB.GetSimApp().AccountKeeper.GetModuleAddress(types.ModuleName).String()
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(pathAtoB.EndpointA.ChannelConfig.PortID, pathAtoB.EndpointA.ChannelID)),
				Amount: "100",
			},
		},
		address,
		receiver.GetAddress().String(),
		"", types.ForwardingPacketData{},
	)

	packet = channeltypes.NewPacket(
		data.GetBytes(),
		1,
		pathBtoC.EndpointA.ChannelConfig.PortID,
		pathBtoC.EndpointA.ChannelID,
		pathBtoC.EndpointB.ChannelConfig.PortID,
		pathBtoC.EndpointB.ChannelID,
		packet.TimeoutHeight,
		packet.TimeoutTimestamp)

	// retrieve module callbacks
	module, _, err := suite.chainB.App.GetIBCKeeper().PortKeeper.LookupModuleByPort(suite.chainB.GetContext(), pathBtoC.EndpointA.ChannelConfig.PortID)
	suite.Require().NoError(err)

	cbs, ok := suite.chainB.App.GetIBCKeeper().PortKeeper.Route(module)
	suite.Require().True(ok)

	// Trigger OnTimeoutPacket for chainB
	err = cbs.OnTimeoutPacket(suite.chainB.GetContext(), packet, nil)
	suite.Require().NoError(err)

	// Ensure that chainB has an ack.
	storedAck, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
	suite.Require().True(found, "chainB does not have an ack")

	// And that this ack is of the type we expect (Error due to time out)
	ack := channeltypes.NewErrorAcknowledgement(types.ErrForwardedPacketTimedOut)
	ackbytes := channeltypes.CommitAcknowledgement(ack.Acknowledgement())
	suite.Require().Equal(ackbytes, storedAck)

	forwardingPacketData := types.NewForwardingPacketData("", forwarding.Hops...)
	data = types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  types.NewDenom(sdk.DefaultBondDenom),
				Amount: "100",
			},
		},
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		"", forwardingPacketData,
	)

	packet = channeltypes.NewPacket(
		data.GetBytes(),
		1,
		pathAtoB.EndpointA.ChannelConfig.PortID,
		pathAtoB.EndpointA.ChannelID,
		pathAtoB.EndpointB.ChannelConfig.PortID,
		pathAtoB.EndpointB.ChannelID,
		packet.TimeoutHeight,
		packet.TimeoutTimestamp)

	// Send the ack to chain A.
	err = suite.chainA.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainA.GetContext(), packet, data, ack)
	suite.Require().NoError(err)

	// Finally, check that A,B, and C escrow accounts do not have fund.
	suite.assertAmountOnChain(suite.chainC, escrow, sdkmath.NewInt(0), denomABC.IBCDenom())
	suite.assertAmountOnChain(suite.chainB, escrow, sdkmath.NewInt(0), denomAB.IBCDenom())
	suite.assertAmountOnChain(suite.chainA, escrow, sdkmath.NewInt(0), denomA.IBCDenom())

	// And that A has its original balance back.
	suite.assertAmountOnChain(suite.chainA, balance, originalABalance.Amount, coin.Denom)
}
