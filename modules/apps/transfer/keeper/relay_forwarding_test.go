package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"

	internaltypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/internal/types"
	"github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
)

const (
	escrow amountType = iota
	balance
)

type ForwardingTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
	chainD *ibctesting.TestChain
}

type amountType int

func TestForwardingTestSuite(t *testing.T) {
	testifysuite.Run(t, new(ForwardingTestSuite))
}

func (suite *ForwardingTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 4)

	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
	suite.chainD = suite.coordinator.GetChain(ibctesting.GetChainID(4))
}

func (suite *ForwardingTestSuite) setupForwardingPaths() (pathAtoB, pathBtoC *ibctesting.Path) {
	pathAtoB = ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	pathBtoC = ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	pathAtoB.Setup()
	pathBtoC.Setup()

	return pathAtoB, pathBtoC
}

// TestStoredForwardedPacketAndEscrowAfterFirstHop tests that the forwarded packet
// from chain A to chain B is stored after when the packet is received on chain B
// and then forwarded to chain C, and checks the balance of the escrow accounts
// in chain A nad B.
func (suite *ForwardingTestSuite) TestStoredForwardedPacketAndEscrowAfterFirstHop() {
	/*
		Given the following topology:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-0) chain A
		stake                  transfer/channel-0/stake           transfer/channel-0/transfer/channel-0/stake
		We want to trigger:
			1. A sends B over channel-0.
			2. Receive on B.
			At this point we want to assert:
				A: escrowA = amount,stake AND packet A to B is stored in forwarded packet
				B: escrowB = amount,transfer/channel-0/stake
	*/

	amount := sdkmath.NewInt(100)
	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	coin := ibctesting.TestCoin
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainC.SenderAccounts[0].SenderAccount
	forwarding := types.NewForwarding(false, types.NewHop(
		pathBtoC.EndpointA.ChannelConfig.PortID,
		pathBtoC.EndpointA.ChannelID,
	))

	transferMsg := types.NewMsgTransfer(
		pathAtoB.EndpointA.ChannelConfig.PortID,
		pathAtoB.EndpointA.ChannelID,
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

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointB.RecvPacketWithResult(packet)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID, packet.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packet, forwardedPacket)

	suite.assertAmountOnChain(suite.chainA, escrow, amount, sdk.DefaultBondDenom)

	// denom path: transfer/channel-0
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	suite.assertAmountOnChain(suite.chainB, escrow, amount, denom.IBCDenom())
}

// TestSuccessfulForward tests a successful transfer from A to C through B.
func (suite *ForwardingTestSuite) TestSuccessfulForward() {
	/*
		Given the following topology:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-0) chain C
		stake                  transfer/channel-0/stake           transfer/channel-0/transfer/channel-0/stake
		We want to trigger:
			1. A sends B over channel-0.
			2. Receive on B.
				2.1 B sends C over channel-1
			3. Receive on C.
		At this point we want to assert:
			A: escrowA = amount,stake
			B: escrowB = amount,transfer/channel-0/denom
			C: finalReceiver = amount,transfer/channel-0/transfer/channel-0/denom
	*/

	amount := sdkmath.NewInt(100)

	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	coinOnA := ibctesting.TestCoin
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainC.SenderAccounts[0].SenderAccount
	forwarding := types.NewForwarding(false, types.NewHop(
		pathBtoC.EndpointA.ChannelConfig.PortID,
		pathBtoC.EndpointA.ChannelID,
	))

	transferMsg := types.NewMsgTransfer(
		pathAtoB.EndpointA.ChannelConfig.PortID,
		pathAtoB.EndpointA.ChannelID,
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

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointB.RecvPacketWithResult(packetFromAtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow A has amount
	suite.assertAmountOnChain(suite.chainA, escrow, amount, sdk.DefaultBondDenom)

	// denom path: transfer/channel-0
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	suite.assertAmountOnChain(suite.chainB, escrow, amount, denom.IBCDenom())

	packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoC)

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// B should have stored the forwarded packet.
	_, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoC.SourcePort, packetFromBtoC.SourceChannel, packetFromBtoC.Sequence)
	suite.Require().True(found, "Chain B should have stored the forwarded packet")

	result, err = pathBtoC.EndpointB.RecvPacketWithResult(packetFromBtoC)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// transfer/channel-0/transfer/channel-0/denom
	// Check that the final receiver has received the expected tokens.
	denomABC := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID), types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	// Check that the final receiver has received the expected tokens.
	suite.assertAmountOnChain(suite.chainC, balance, amount, denomABC.IBCDenom())

	successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	successAckBz := channeltypes.CommitAcknowledgement(successAck.Acknowledgement())
	ackOnC := suite.chainC.GetAcknowledgement(packetFromBtoC)
	suite.Require().Equal(successAckBz, ackOnC)

	// Ack back to B
	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointA.AcknowledgePacket(packetFromBtoC, successAck.Acknowledgement())
	suite.Require().NoError(err)

	ackOnB := suite.chainB.GetAcknowledgement(packetFromAtoB)
	suite.Require().Equal(successAckBz, ackOnB)

	// B should now have deleted the forwarded packet.
	_, found = suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoC.SourcePort, packetFromBtoC.SourceChannel, packetFromBtoC.Sequence)
	suite.Require().False(found, "Chain B should have deleted the forwarded packet")

	// Ack back to A
	err = pathAtoB.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathAtoB.EndpointA.AcknowledgePacket(packetFromAtoB, successAck.Acknowledgement())
	suite.Require().NoError(err)
}

// TestSuccessfulForwardWithMemo tests a successful transfer from A to C through B with a memo that should arrive at C.
func (suite *ForwardingTestSuite) TestSuccessfulForwardWithMemo() {
	/*
		Given the following topology:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-0) chain C
		stake                  transfer/channel-0/stake           transfer/channel-0/transfer/channel-0/stake
		We want to trigger:
			1. A sends B over channel-0.
			2. Receive on B.
				2.1 B sends C over channel-1
			3. Receive on C.
		At this point we want to assert:
			A: escrowA = amount,stake
			B: escrowB = amount,transfer/channel-0/denom
			C: finalReceiver = amount,transfer/channel-0/transfer/channel-0/denom,memo
	*/

	amount := sdkmath.NewInt(100)
	testMemo := "test forwarding memo"

	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	coinOnA := ibctesting.TestCoin
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainC.SenderAccounts[0].SenderAccount
	forwarding := types.NewForwarding(false, types.NewHop(
		pathBtoC.EndpointA.ChannelConfig.PortID,
		pathBtoC.EndpointA.ChannelID,
	))

	transferMsg := types.NewMsgTransfer(
		pathAtoB.EndpointA.ChannelConfig.PortID,
		pathAtoB.EndpointA.ChannelID,
		sdk.NewCoins(coinOnA),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(),
		testMemo,
		forwarding,
	)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainB
	packetFromAtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromAtoB)

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// Check that the memo is stored correctly in the packet sent from A
	var tokenPacketOnA types.FungibleTokenPacketDataV2
	err = proto.Unmarshal(packetFromAtoB.Data, &tokenPacketOnA)
	suite.Require().NoError(err)
	suite.Require().Equal("", tokenPacketOnA.Memo)
	suite.Require().Equal(testMemo, tokenPacketOnA.Forwarding.DestinationMemo)

	result, err = pathAtoB.EndpointB.RecvPacketWithResult(packetFromAtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow A has amount
	suite.assertAmountOnChain(suite.chainA, escrow, amount, sdk.DefaultBondDenom)

	// denom path: transfer/channel-0
	denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	suite.assertAmountOnChain(suite.chainB, escrow, amount, denom.IBCDenom())

	packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoC)

	// Check that the memo is stored correctly in the packet sent from B
	var tokenPacketOnB types.FungibleTokenPacketDataV2
	err = proto.Unmarshal(packetFromBtoC.Data, &tokenPacketOnB)
	suite.Require().NoError(err)
	suite.Require().Equal(testMemo, tokenPacketOnB.Memo)
	suite.Require().Equal("", tokenPacketOnB.Forwarding.DestinationMemo)

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// B should have stored the forwarded packet.
	_, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoC.SourcePort, packetFromBtoC.SourceChannel, packetFromBtoC.Sequence)
	suite.Require().True(found, "Chain B should have stored the forwarded packet")

	result, err = pathBtoC.EndpointB.RecvPacketWithResult(packetFromBtoC)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	packetOnC, err := ibctesting.ParseRecvPacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetOnC)

	// Check that the memo is stored directly in the memo field on C
	var tokenPacketOnC types.FungibleTokenPacketDataV2
	err = proto.Unmarshal(packetOnC.Data, &tokenPacketOnC)
	suite.Require().NoError(err)
	suite.Require().Equal("", tokenPacketOnC.Forwarding.DestinationMemo)
	suite.Require().Equal(testMemo, tokenPacketOnC.Memo)

	// transfer/channel-0/transfer/channel-0/denom
	// Check that the final receiver has received the expected tokens.
	denomABC := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID), types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	// Check that the final receiver has received the expected tokens.
	suite.assertAmountOnChain(suite.chainC, balance, amount, denomABC.IBCDenom())

	successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	successAckBz := channeltypes.CommitAcknowledgement(successAck.Acknowledgement())
	ackOnC := suite.chainC.GetAcknowledgement(packetFromBtoC)
	suite.Require().Equal(successAckBz, ackOnC)

	// Ack back to B
	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointA.AcknowledgePacket(packetFromBtoC, successAck.Acknowledgement())
	suite.Require().NoError(err)

	ackOnB := suite.chainB.GetAcknowledgement(packetFromAtoB)
	suite.Require().Equal(successAckBz, ackOnB)

	// B should now have deleted the forwarded packet.
	_, found = suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoC.SourcePort, packetFromBtoC.SourceChannel, packetFromBtoC.Sequence)
	suite.Require().False(found, "Chain B should have deleted the forwarded packet")

	// Ack back to A
	err = pathAtoB.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathAtoB.EndpointA.AcknowledgePacket(packetFromAtoB, successAck.Acknowledgement())
	suite.Require().NoError(err)
}

// TestSuccessfulForwardWithNonCosmosAccAddress tests that a packet is successfully forwarded with a non-Cosmos account address.
// The test stops before verifying the final receive, because we don't have a non-cosmos chain to test with.
func (suite *ForwardingTestSuite) TestSuccessfulForwardWithNonCosmosAccAddress() {
	/*
		Given the following topology:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-0) chain C
		stake                  transfer/channel-0/stake           transfer/channel-0/transfer/channel-0/stake
		We want to trigger:
			1. A sends B over channel-0.
			2. Receive on B.
				2.1 B sends C over channel-1
			3. Receive on C.
		At this point we want to assert:
			A: packet gets relayed properly with final receiver intact
			B: packet arrives and is forwarded with final receiver intact
			C: no assertions as we don't have a non-cosmos chain to test with, so this would fail here
	*/

	amount := sdkmath.NewInt(100)

	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	sender := suite.chainA.SenderAccounts[0].SenderAccount
	nonCosmosReceiver := "0x42069163Ac5919fD49e6A67e6c211E0C86397fa2"
	forwarding := types.NewForwarding(false, types.NewHop(
		pathBtoC.EndpointA.ChannelConfig.PortID,
		pathBtoC.EndpointA.ChannelID,
	))

	transferMsg := types.NewMsgTransfer(
		pathAtoB.EndpointA.ChannelConfig.PortID,
		pathAtoB.EndpointA.ChannelID,
		sdk.NewCoins(ibctesting.TestCoin),
		sender.GetAddress().String(),
		nonCosmosReceiver,
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

	// Check that the token sent from A has final receiver intact
	var tokenPacketOnA types.FungibleTokenPacketDataV2
	err = proto.Unmarshal(packetFromAtoB.Data, &tokenPacketOnA)
	suite.Require().NoError(err)
	suite.Require().Equal(nonCosmosReceiver, tokenPacketOnA.Receiver)

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointB.RecvPacketWithResult(packetFromAtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow A has amount
	suite.assertAmountOnChain(suite.chainA, escrow, amount, sdk.DefaultBondDenom)

	packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoC)

	// Check that the token sent from B has final receiver intact
	var tokenPacketOnB types.FungibleTokenPacketDataV2
	err = proto.Unmarshal(packetFromBtoC.Data, &tokenPacketOnB)
	suite.Require().NoError(err)
	suite.Require().Equal(nonCosmosReceiver, tokenPacketOnB.Receiver)

	// B should have stored the forwarded packet.
	_, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoC.SourcePort, packetFromBtoC.SourceChannel, packetFromBtoC.Sequence)
	suite.Require().True(found, "Chain B should have stored the forwarded packet")
}

// TestSuccessfulUnwind tests unwinding of tokens sent from A -> B -> C by
// forwarding the tokens back from C -> B -> A.
func (suite *ForwardingTestSuite) TestSuccessfulUnwind() {
	/*
		Given the following topolgy:
		chain A (channel 0) -> (channel-0) chain B (channel-1) -> (channel-0) chain C
		stake                  transfer/channel-0/stake           transfer/channel-0/transfer/channel-0/stake
		We want to trigger:
			1. Send vouchers from C to B.
			2. Receive on B.
				2.1 B sends B over channel-0
			3. Receive on A.
			At this point we want to assert:
				- escrow on B and C is zero
				- receiver on A has amount,stake
	*/

	amount := sdkmath.NewInt(100)

	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	sender := suite.chainC.SenderAccount
	receiver := suite.chainA.SenderAccount

	// set sender and escrow accounts with the right balances to set up an initial state
	// that should have been the same as sending token from A -> B -> C
	denomA := types.NewDenom(sdk.DefaultBondDenom)
	denomAB := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	denomABC := types.NewDenom(sdk.DefaultBondDenom, append([]types.Hop{types.NewHop(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID)}, denomAB.Trace...)...)

	coinOnA := sdk.NewCoin(denomA.IBCDenom(), amount)
	err := suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(coinOnA))
	suite.Require().NoError(err)
	escrowAddressAtoB := types.GetEscrowAddress(pathAtoB.EndpointA.ChannelConfig.PortID, pathAtoB.EndpointA.ChannelID)
	err = suite.chainA.GetSimApp().BankKeeper.MintCoins(suite.chainA.GetContext(), types.ModuleName, sdk.NewCoins(coinOnA))
	suite.Require().NoError(err)
	err = suite.chainA.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainA.GetContext(), types.ModuleName, escrowAddressAtoB, sdk.NewCoins(coinOnA))
	suite.Require().NoError(err)
	suite.chainA.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainA.GetContext(), coinOnA)

	coinOnB := sdk.NewCoin(denomAB.IBCDenom(), amount)
	err = suite.chainB.GetSimApp().BankKeeper.MintCoins(suite.chainB.GetContext(), types.ModuleName, sdk.NewCoins(coinOnB))
	suite.Require().NoError(err)
	escrowAddressBtoC := types.GetEscrowAddress(pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID)
	err = suite.chainB.GetSimApp().BankKeeper.MintCoins(suite.chainB.GetContext(), types.ModuleName, sdk.NewCoins(coinOnB))
	suite.Require().NoError(err)
	err = suite.chainB.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainB.GetContext(), types.ModuleName, escrowAddressBtoC, sdk.NewCoins(coinOnB))
	suite.Require().NoError(err)
	suite.chainB.GetSimApp().TransferKeeper.SetTotalEscrowForDenom(suite.chainB.GetContext(), coinOnB)
	suite.chainB.GetSimApp().TransferKeeper.SetDenom(suite.chainB.GetContext(), denomAB)

	coinOnC := sdk.NewCoin(denomABC.IBCDenom(), amount)
	err = suite.chainC.GetSimApp().BankKeeper.MintCoins(suite.chainC.GetContext(), types.ModuleName, sdk.NewCoins(coinOnC))
	suite.Require().NoError(err)
	err = suite.chainC.GetSimApp().BankKeeper.SendCoinsFromModuleToAccount(suite.chainC.GetContext(), types.ModuleName, sender.GetAddress(), sdk.NewCoins(coinOnC))
	suite.Require().NoError(err)
	suite.chainC.GetSimApp().TransferKeeper.SetDenom(suite.chainC.GetContext(), denomABC)

	originalABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), receiver.GetAddress(), coinOnA.Denom)

	forwarding := types.NewForwarding(false, types.NewHop(
		pathAtoB.EndpointB.ChannelConfig.PortID,
		pathAtoB.EndpointB.ChannelID,
	))

	transferMsg := types.NewMsgTransfer(
		pathBtoC.EndpointB.ChannelConfig.PortID,
		pathBtoC.EndpointB.ChannelID,
		sdk.NewCoins(coinOnC),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainC.GetTimeoutTimestamp(), "",
		forwarding,
	)

	result, err := suite.chainC.SendMsgs(transferMsg)
	suite.Require().NoError(err) // message committed

	// Sender's balance for vouchers is 0
	suite.assertAmountOnChain(suite.chainC, balance, sdkmath.NewInt(0), denomABC.IBCDenom())

	// parse the packet from result events and recv packet on chainB
	packetFromCtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromCtoB)

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathBtoC.EndpointA.RecvPacketWithResult(packetFromCtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Vouchers have been burned on chain B
	suite.assertAmountOnChain(suite.chainB, escrow, sdkmath.NewInt(0), denomAB.IBCDenom())

	// parse the packet from result events and recv packet on chainA
	packetFromBtoA, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoA)

	err = pathAtoB.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	// B should have stored the forwarded packet.
	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoA.SourcePort, packetFromBtoA.SourceChannel, packetFromBtoA.Sequence)
	suite.Require().True(found)
	suite.Require().Equal(packetFromCtoB, forwardedPacket)

	result, err = pathAtoB.EndpointA.RecvPacketWithResult(packetFromBtoA)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	successAckBz := channeltypes.CommitAcknowledgement(successAck.Acknowledgement())

	// Ack back to B
	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = pathAtoB.EndpointB.AcknowledgePacket(packetFromBtoA, successAck.Acknowledgement())
	suite.Require().NoError(err)

	ackOnB := suite.chainB.GetAcknowledgement(packetFromCtoB)
	suite.Require().Equal(successAckBz, ackOnB)

	// Ack back to C
	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointB.AcknowledgePacket(packetFromCtoB, successAck.Acknowledgement())
	suite.Require().NoError(err)

	// Check that B deleted the forwarded packet.
	_, found = suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoA.SourcePort, packetFromBtoA.SourceChannel, packetFromBtoA.Sequence)
	suite.Require().False(found, "chain B should have deleted the forwarded packet mapping")

	// Check that tokens have been unescrowed and receiver got the tokens
	suite.assertAmountOnChain(suite.chainA, escrow, sdkmath.NewInt(0), denomA.IBCDenom())
	suite.assertAmountOnChain(suite.chainA, balance, originalABalance.Amount.Add(amount), denomA.IBCDenom())
}

// TestAcknowledgementFailureWithMiddleChainAsNativeTokenSource tests a failure in the last hop where the
// middle chain is native source when receiving and sending the packet. In other words, the middle chain's native
// token has been sent to chain C, and the multi-hop transfer from C -> B -> A has chain B being the source of
// the token both when receiving and forwarding (sending).
func (suite *ForwardingTestSuite) TestAcknowledgementFailureWithMiddleChainAsNativeTokenSource() {
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

	amount := sdkmath.NewInt(100)

	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	coinOnB := ibctesting.TestCoin
	setupSender := suite.chainB.SenderAccounts[0].SenderAccount
	setupReceiver := suite.chainC.SenderAccounts[0].SenderAccount

	setupTransferMsg := types.NewMsgTransfer(
		pathBtoC.EndpointA.ChannelConfig.PortID,
		pathBtoC.EndpointA.ChannelID,
		sdk.NewCoins(coinOnB),
		setupSender.GetAddress().String(),
		setupReceiver.GetAddress().String(),
		suite.chainB.GetTimeoutHeight(),
		0, "",
		nil,
	)

	result, err := suite.chainB.SendMsgs(setupTransferMsg)
	suite.Require().NoError(err) // message committed

	// parse the packet from result events and recv packet on chainC
	packetFromBToC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBToC)

	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathBtoC.EndpointB.RecvPacketWithResult(packetFromBToC)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that EscrowBtoC has amount
	escrowAddressBtoC := types.GetEscrowAddress(pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID)
	escrowBalancBtoC := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoC, coinOnB.GetDenom())
	suite.Require().Equal(amount, escrowBalancBtoC.Amount)

	// Check that receiver has the expected tokens
	denomOnC := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID))
	coinOnC := sdk.NewCoin(denomOnC.IBCDenom(), amount)
	balanceOnC := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), setupReceiver.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(amount, balanceOnC.Amount)

	// Now we start the transfer from C -> B -> A
	sender := suite.chainC.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[0].SenderAccount

	forwarding := types.NewForwarding(false, types.NewHop(
		pathAtoB.EndpointB.ChannelConfig.PortID,
		pathAtoB.EndpointB.ChannelID,
	))

	forwardTransfer := types.NewMsgTransfer(
		pathBtoC.EndpointB.ChannelConfig.PortID,
		pathBtoC.EndpointB.ChannelID,
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

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathBtoC.EndpointA.RecvPacketWithResult(packetFromCtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that escrow has been moved from EscrowBtoC to EscrowBtoA
	escrowBalancBtoC = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoC, coinOnB.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), escrowBalancBtoC.Amount)

	escrowAddressBtoA := types.GetEscrowAddress(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID)
	escrowBalanceBtoA := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoA, coinOnB.GetDenom())
	suite.Require().Equal(amount, escrowBalanceBtoA.Amount)

	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID, packetFromCtoB.Sequence)
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

	err = pathAtoB.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointA.RecvPacketWithResult(packetFromBtoA)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	ack, err := ibctesting.ParseAckFromEvents(result.Events)
	suite.Require().NoError(err)

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointB.AcknowledgePacketWithResult(packetFromBtoA, ack)
	suite.Require().NoError(err)

	ack, err = ibctesting.ParseAckFromEvents(result.Events)
	suite.Require().NoError(err)

	// Check that escrow has been moved back from EscrowBtoA to EscrowBtoC
	escrowBalanceBtoA = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoA, coinOnB.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), escrowBalanceBtoA.Amount)

	escrowBalancBtoC = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddressBtoC, coinOnB.GetDenom())
	suite.Require().Equal(amount, escrowBalancBtoC.Amount)

	// Check the status of account on chain C before executing ack.
	balanceOnC = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), setupReceiver.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(sdkmath.NewInt(0), balanceOnC.Amount)

	// Propagate the error to C
	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointB.AcknowledgePacket(packetFromCtoB, ack)
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
func (suite *ForwardingTestSuite) TestAcknowledgementFailureWithMiddleChainAsNotBeingTokenSource() {
	/*
		Given the following topology:
		chain A (channel 0) 												<- (channel-0) chain B (channel-1) <- (channel-0) chain C
		transfer/channel-0/transfer/channel-1/stake    transfer/channel-1/stake           stake
		We want to trigger:
			1. Single transfer forwarding token from C -> B -> A
				1.1 The ack fails on the last hop
				1.2 Propagate the error back to C
			2. Verify all the balances are updated as expected
	*/

	amount := sdkmath.NewInt(100)

	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	// Now we start the transfer from C -> B -> A
	coinOnC := ibctesting.TestCoin
	sender := suite.chainC.SenderAccounts[0].SenderAccount
	receiver := suite.chainA.SenderAccounts[0].SenderAccount

	forwarding := types.NewForwarding(false, types.NewHop(
		pathAtoB.EndpointB.ChannelConfig.PortID,
		pathAtoB.EndpointB.ChannelID,
	))

	forwardTransfer := types.NewMsgTransfer(
		pathBtoC.EndpointB.ChannelConfig.PortID,
		pathBtoC.EndpointB.ChannelID,
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

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathBtoC.EndpointA.RecvPacketWithResult(packetFromCtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow B has amount
	denomOnB := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID))
	suite.assertAmountOnChain(suite.chainB, escrow, amount, denomOnB.IBCDenom())

	forwardedPacket, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID, packetFromCtoB.Sequence)
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

	err = pathAtoB.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointA.RecvPacketWithResult(packetFromBtoA)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	ack, err := ibctesting.ParseAckFromEvents(result.Events)
	suite.Require().NoError(err)

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointB.AcknowledgePacketWithResult(packetFromBtoA, ack)
	suite.Require().NoError(err)

	ack, err = ibctesting.ParseAckFromEvents(result.Events)
	suite.Require().NoError(err)

	// Check that escrow has been burnt on B
	suite.assertAmountOnChain(suite.chainB, escrow, sdkmath.NewInt(0), denomOnB.IBCDenom())

	// Check the status of account on chain C before executing ack.
	balanceOnC := suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), sender.GetAddress(), coinOnC.GetDenom())
	suite.Require().Equal(balanceOnCBefore.SubAmount(amount).Amount, balanceOnC.Amount)

	// Propagate the error to C
	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointB.AcknowledgePacket(packetFromCtoB, ack)
	suite.Require().NoError(err)

	// Check that everything has been reverted
	//
	// Check the token has been returned to the sender on C
	suite.assertAmountOnChain(suite.chainC, escrow, sdkmath.NewInt(0), coinOnC.GetDenom())
	suite.assertAmountOnChain(suite.chainC, balance, balanceOnCBefore.Amount, coinOnC.GetDenom())
}

// TestOnTimeoutPacketForwarding tests the scenario in which a packet goes from
// A to C, using B as a forwarding hop. The packet times out when going to C
// from B and we verify that funds are properly returned to A.
func (suite *ForwardingTestSuite) TestOnTimeoutPacketForwarding() {
	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	amount := sdkmath.NewInt(100)
	coin := ibctesting.TestCoin
	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainC.SenderAccounts[0].SenderAccount

	denomA := types.NewDenom(coin.Denom)
	denomAB := types.NewDenom(coin.Denom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	denomABC := types.NewDenom(coin.Denom, append([]types.Hop{types.NewHop(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID)}, denomAB.Trace...)...)

	originalABalance := suite.chainA.GetSimApp().BankKeeper.GetBalance(suite.chainA.GetContext(), sender.GetAddress(), coin.Denom)

	forwarding := types.NewForwarding(false, types.NewHop(pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID))

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
				Denom:  types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID)),
				Amount: "100",
			},
		},
		address,
		receiver.GetAddress().String(),
		"", ibctesting.EmptyForwardingPacketData,
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
	storedAck, found := suite.chainB.App.GetIBCKeeper().ChannelKeeper.GetPacketAcknowledgement(suite.chainB.GetContext(), pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID, packet.GetSequence())
	suite.Require().True(found, "chainB does not have an ack")

	// And that this ack is of the type we expect (Error due to time out)
	ack := internaltypes.NewForwardTimeoutAcknowledgement(packet)
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

// TestForwardingWithMoreThanOneHop tests the scenario in which we
// forward with more than one forwarding hop.
func (suite *ForwardingTestSuite) TestForwardingWithMoreThanOneHop() {
	// Setup A->B->C->D
	coinOnA := ibctesting.TestCoin

	pathAtoB := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	pathAtoB.Setup()

	pathBtoC := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	pathBtoC.Setup()

	pathCtoD := ibctesting.NewTransferPath(suite.chainC, suite.chainD)
	pathCtoD.Setup()

	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainD.SenderAccounts[0].SenderAccount

	forwarding := types.NewForwarding(false,
		types.NewHop(pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID),
		types.NewHop(pathCtoD.EndpointA.ChannelConfig.PortID, pathCtoD.EndpointA.ChannelID),
	)

	transferMsg := types.NewMsgTransfer(
		pathAtoB.EndpointA.ChannelConfig.PortID,
		pathAtoB.EndpointA.ChannelID,
		sdk.NewCoins(coinOnA),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(),
		"",
		forwarding)

	// Send message to A and verify.
	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err)

	packetFromAtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromAtoB)

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// Receive from B and verify.
	result, err = pathAtoB.EndpointB.RecvPacketWithResult(packetFromAtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow A has amount
	suite.assertAmountOnChain(suite.chainA, escrow, coinOnA.Amount, coinOnA.Denom)

	// Check that Escrow B has amount
	denomTrace := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	suite.assertAmountOnChain(suite.chainB, escrow, coinOnA.Amount, denomTrace.IBCDenom())

	// Receive on C the packet sent from B, verify amount.
	packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoC)

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathBtoC.EndpointB.RecvPacketWithResult(packetFromBtoC)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// Check that Escrow C has amount
	denomTraceABC := types.NewDenom(denomTrace.Base, append([]types.Hop{types.NewHop(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID)}, denomTrace.Trace...)...)
	suite.assertAmountOnChain(suite.chainC, escrow, coinOnA.Amount, denomTraceABC.IBCDenom())

	// Finally, receive on D and verify that D has the desired amount.
	packetFromCtoD, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromCtoD)

	err = pathCtoD.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathCtoD.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathCtoD.EndpointB.RecvPacketWithResult(packetFromCtoD)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	denomTraceABCD := types.NewDenom(denomTraceABC.Base, append([]types.Hop{types.NewHop(pathCtoD.EndpointB.ChannelConfig.PortID, pathCtoD.EndpointB.ChannelID)}, denomTraceABC.Trace...)...)
	suite.assertAmountOnChain(suite.chainD, balance, coinOnA.Amount, denomTraceABCD.IBCDenom())

	// Propagate the ack back from D to A.
	ack, err := ibctesting.ParseAckFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(ack)

	err = pathCtoD.EndpointA.AcknowledgePacket(packetFromCtoD, ack)
	suite.Require().NoError(err)

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointA.AcknowledgePacket(packetFromBtoC, ack)
	suite.Require().NoError(err)

	err = pathAtoB.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathAtoB.EndpointA.AcknowledgePacket(packetFromAtoB, ack)
	suite.Require().NoError(err)
}

func (suite *ForwardingTestSuite) TestMultihopForwardingErrorAcknowledgement() {
	// Setup A->B->C->D
	coinOnA := ibctesting.TestCoin

	pathAtoB := ibctesting.NewTransferPath(suite.chainA, suite.chainB)
	pathAtoB.Setup()

	pathBtoC := ibctesting.NewTransferPath(suite.chainB, suite.chainC)
	pathBtoC.Setup()

	pathCtoD := ibctesting.NewTransferPath(suite.chainC, suite.chainD)
	pathCtoD.Setup()

	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainD.SenderAccounts[0].SenderAccount

	forwarding := types.NewForwarding(false,
		types.NewHop(pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID),
		types.NewHop(pathCtoD.EndpointA.ChannelConfig.PortID, pathCtoD.EndpointA.ChannelID),
	)

	transferMsg := types.NewMsgTransfer(
		pathAtoB.EndpointA.ChannelConfig.PortID,
		pathAtoB.EndpointA.ChannelID,
		sdk.NewCoins(coinOnA),
		sender.GetAddress().String(),
		receiver.GetAddress().String(),
		clienttypes.ZeroHeight(),
		suite.chainA.GetTimeoutTimestamp(),
		"",
		forwarding)

	result, err := suite.chainA.SendMsgs(transferMsg)
	suite.Require().NoError(err)

	packetFromAtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromAtoB)

	err = pathAtoB.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointB.RecvPacketWithResult(packetFromAtoB)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// assert escrow on chain A.
	suite.assertAmountOnChain(suite.chainA, escrow, coinOnA.Amount, coinOnA.Denom)

	// assert escrow on chain B.
	denomAB := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	suite.assertAmountOnChain(suite.chainB, escrow, coinOnA.Amount, denomAB.IBCDenom())

	packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromBtoC)

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathBtoC.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathBtoC.EndpointB.RecvPacketWithResult(packetFromBtoC)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// assert escrow on chain C.
	denomABC := types.NewDenom(denomAB.Base, append([]types.Hop{types.NewHop(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID)}, denomAB.Trace...)...)
	suite.assertAmountOnChain(suite.chainC, escrow, coinOnA.Amount, denomABC.IBCDenom())

	packetFromCtoD, err := ibctesting.ParsePacketFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(packetFromCtoD)

	err = pathCtoD.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	err = pathCtoD.EndpointB.UpdateClient()
	suite.Require().NoError(err)

	// force an error acknowledgement by disabling the receive param on chain D.
	ctx := pathCtoD.EndpointB.Chain.GetContext()
	pathCtoD.EndpointB.Chain.GetSimApp().TransferKeeper.SetParams(ctx, types.NewParams(true, false))

	result, err = pathCtoD.EndpointB.RecvPacketWithResult(packetFromCtoD)
	suite.Require().NoError(err)
	suite.Require().NotNil(result)

	// propagate the acknowledgement from chain D to chain A.
	ack, err := ibctesting.ParseAckFromEvents(result.Events)
	suite.Require().NoError(err)
	suite.Require().NotNil(ack)

	result, err = pathCtoD.EndpointA.AcknowledgePacketWithResult(packetFromCtoD, ack)
	suite.Require().NoError(err)

	ack, err = ibctesting.ParseAckFromEvents(result.Events)
	suite.Require().NoError(err)

	err = pathBtoC.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathBtoC.EndpointA.AcknowledgePacketWithResult(packetFromBtoC, ack)
	suite.Require().NoError(err)

	ack, err = ibctesting.ParseAckFromEvents(result.Events)
	suite.Require().NoError(err)

	err = pathAtoB.EndpointA.UpdateClient()
	suite.Require().NoError(err)

	result, err = pathAtoB.EndpointA.AcknowledgePacketWithResult(packetFromAtoB, ack)
	suite.Require().NoError(err)

	// NOTE: parse acknowledgement from transfer events as ack is not emitted in core AcknowledgePacket events.
	ackStr, err := parseAckFromTransferEvents(result.Events)
	suite.Require().NoError(err)

	expected := fmt.Sprintf(`error:"forwarding packet failed on %s/%s: forwarding packet failed on %s/%s: ABCI code: 8: error handling packet: see events for details" `, pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID, pathCtoD.EndpointA.ChannelConfig.PortID, pathCtoD.EndpointA.ChannelID)
	suite.Require().Equal(expected, ackStr)
}

func parseAckFromTransferEvents(events []abci.Event) (string, error) {
	for _, ev := range events {
		if ev.Type == types.EventTypePacket {
			for _, attr := range ev.Attributes {
				if attr.Key == types.AttributeKeyAck {
					return attr.Value, nil
				}
			}
		}
	}

	return "", fmt.Errorf("acknowledgement event attribute not found")
}

func (suite *ForwardingTestSuite) assertAmountOnChain(chain *ibctesting.TestChain, balanceType amountType, amount sdkmath.Int, denom string) {
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
