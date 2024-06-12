package keeper_test

import (
	"fmt"

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
	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path2.EndpointA.ChannelConfig.PortID,
				ChannelId: path2.EndpointA.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
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
	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path2.EndpointB.ChannelConfig.PortID,
				ChannelId: path2.EndpointB.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
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
	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path2.EndpointB.ChannelConfig.PortID,
				ChannelId: path2.EndpointB.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	denom := types.Denom{Base: sdk.DefaultBondDenom}
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, sender.GetAddress().String(), receiver.GetAddress().String(), "", &forwardingPath)
	packetRecv := channeltypes.NewPacket(data.GetBytes(), 2, path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, clienttypes.NewHeight(1, 100), 0)

	var async bool
	async, err = suite.chainB.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainB.GetContext(), packetRecv, data)
	// If forwarding has been triggered then the async must be true.
	suite.Require().True(async)
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
		}, types.GetForwardAddress(path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID).String(), receiver.GetAddress().String(), "", nil)
	packetRecv = channeltypes.NewPacket(data.GetBytes(), 3, path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID, path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)

	// execute onRecvPacket, when chaninA receives the tokens the escrow amount on B should increase to amount
	async, err = suite.chainA.GetSimApp().TransferKeeper.OnRecvPacket(suite.chainA.GetContext(), packetRecv, data)
	suite.Require().False(async)
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
	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path2.EndpointB.ChannelConfig.PortID,
				ChannelId: path2.EndpointB.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg := types.NewMsgTransfer(
		path1.EndpointA.ChannelConfig.PortID,
		path1.EndpointA.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
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

	// Check that Escrow A has amount
	totalEscrowChainA := suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainA.Amount)

	// denomTrace path: transfer/channel-0
	denomTrace := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that Escrow B has amount
	coin = sdk.NewCoin(denomTrace.IBCDenom(), amount)
	totalEscrowChainB := suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path2.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointA.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// transfer/channel-1/transfer/channel-0/denom
	denomTraceABA := types.NewDenom(sdk.DefaultBondDenom, types.NewTrace(path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID), types.NewTrace(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID))

	// Check that the final receiver has received the expected tokens.
	coin = sdk.NewCoin(denomTraceABA.IBCDenom(), amount)
	postCoinOnA := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), suite.chainA.SenderAccounts[1].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), postCoinOnA.Amount, "final receiver balance has not increased")
}

// This test replicates the Acknowledgement Failure Scenario 5
// Currently seems like the middle hop is not reverting state changes when an error occurs.
// In turn the final hop properly reverts changes. There may be an error in the way async ack are managed
// or in the way i'm trying to activate the OnAck function.
func (suite *KeeperTestSuite) TestAcknowledgementFailureScenario5Forwarding() {
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
		nil,
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
		nil,
	)

	result, err = suite.chainB.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointB.RecvPacketWithResult(packet)
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
	// coin = sdk.NewCoin(denomTraceBC.IBCDenom(), amount)

	sender = suite.chainC.SenderAccounts[0].SenderAccount
	receiver = suite.chainA.SenderAccounts[0].SenderAccount // Receiver is the A chain account

	forwardingPath := types.ForwardingInfo{
		Hops: []*types.Hop{
			{
				PortId:    path1.EndpointB.ChannelConfig.PortID,
				ChannelId: path1.EndpointB.ChannelID,
			},
		},
		Memo: "",
	}

	transferMsg = types.NewMsgTransfer(
		path2.EndpointB.ChannelConfig.PortID,
		path2.EndpointB.ChannelID,
		sdk.NewCoins(coin),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		suite.chainA.GetTimeoutHeight(),
		0, "",
		&forwardingPath,
	)

	result, err = suite.chainC.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// Voucher have been burned on chain C
	postCoinOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnC.Amount, "Vouchers have not been burned")

	// parse the packet from result events and recv packet on chainB
	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	err = path2.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = path2.EndpointA.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// We have successfully received the packet on B and forwarded it to A.
	// Lets try to retrieve it in order to save it
	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, packet.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packet, forwardedPacket)

	// Voucher have been burned on chain B
	coin = sdk.NewCoin(denomAB.IBCDenom(), amount)
	postCoinOnB = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnB.Amount, "Vouchers have not been burned")

	// Now we can receive the packet on A.
	// To trigger an error during the OnRecv, we have to manipulate the balance present in the escrow of A
	// of denom

	// parse the packet from result events and recv packet on chainA
	packet, err = ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packet)

	// manipulate escrow account for denom on chain A
	coin = sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(99))
	suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), coin)
	totalEscrowChainA = suite.chainA.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainA.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(99), totalEscrowChainA.Amount)

	err = path1.EndpointA.UpdateClient()
	suite.Require().NoError(err)
	// suite.Require().Equal(packet, forwardedPacket)

	result, err = path1.EndpointA.RecvPacketWithResult(packet)
	suite.Require().Error(err)
	suite.Require().Nil(result)
	// In theory now an error ack should have been written on chain A
	// NOW WE HAVE TO SEND ACK TO B, PROPAGTE ACK TO C, CHECK FINAL RESULTS

	// Reconstruct packet data
	denom := types.ExtractDenomFromPath(denomAB.Path())
	data := types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, types.GetForwardAddress(path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID).String(), suite.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(), "", nil)
	packetRecv := channeltypes.NewPacket(data.GetBytes(), 3, path1.EndpointB.ChannelConfig.PortID, path1.EndpointB.ChannelID, path1.EndpointA.ChannelConfig.PortID, path1.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)

	err = path1.EndpointB.UpdateClient()
	suite.Require().NoError(err)
	ack := channeltypes.NewErrorAcknowledgement(fmt.Errorf("failed packet transfer"))

	// err = path1.EndpointA.AcknowledgePacket(packetRecv, ack.Acknowledgement())
	err = suite.chainB.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainB.GetContext(), packetRecv, data, ack)
	suite.Require().NoError(err)

	// Check that Escrow B has been refunded amount
	// NOTE This is failing. The revertInFlightsChanges sohuld mint back voucher to chainBescrow
	// but this is not happening. It may be a problem related with how we're writing async acks.
	//
	coin = sdk.NewCoin(denomAB.IBCDenom(), amount)
	totalEscrowChainB = suite.chainB.GetSimApp().TransferKeeper.GetTotalEscrowForDenom(suite.chainB.GetContext(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(100), totalEscrowChainB.Amount)

	denom = types.ExtractDenomFromPath(denomABC.Path())
	data = types.NewFungibleTokenPacketDataV2(
		[]types.Token{
			{
				Denom:  denom,
				Amount: amount.String(),
			},
		}, suite.chainC.SenderAccounts[0].SenderAccount.GetAddress().String(), suite.chainA.SenderAccounts[0].SenderAccount.GetAddress().String(), "", nil)
	// suite.chainC.SenderAccounts[0].SenderAccount.GetAddress().String() This should be forward account of B
	packet = channeltypes.NewPacket(data.GetBytes(), 3, path2.EndpointB.ChannelConfig.PortID, path2.EndpointB.ChannelID, path2.EndpointA.ChannelConfig.PortID, path2.EndpointA.ChannelID, clienttypes.NewHeight(1, 100), 0)

	err = path2.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// Check the status of account on chain C before executing ack.
	coin = sdk.NewCoin(denomABC.IBCDenom(), amount)
	postCoinOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccounts[0].SenderAccount.GetAddress(), coin.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), postCoinOnC.Amount, "Final Hop balance has been refunded before Ack execution")

	// Execute ack
	err = suite.chainC.GetSimApp().TransferKeeper.OnAcknowledgementPacket(suite.chainC.GetContext(), packet, data, ack)
	// err = path2.EndpointB.AcknowledgePacket(packet, ack.Acknowledgement())
	suite.Require().NoError(err)

	// Check that everythig has been reverted
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
