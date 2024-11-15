package keeper_test

import (
	"testing"

	testifysuite "github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v9/modules/apps/transfer/types"
	typesv2 "github.com/cosmos/ibc-go/v9/modules/apps/transfer/v2/types"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
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
	pathAtoB.SetupV2()
	pathBtoC.SetupV2()

	return pathAtoB, pathBtoC
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

	pathAtoB, pathBtoC := suite.setupForwardingPaths()

	sender := suite.chainA.SenderAccounts[0].SenderAccount
	receiver := suite.chainC.SenderAccounts[0].SenderAccount

	msg := &typesv2.MsgTransfer{
		SourceChannel:    pathAtoB.EndpointA.ClientID,
		SourcePort:       pathAtoB.EndpointA.ChannelConfig.PortID,
		DestinationPort:  pathAtoB.EndpointB.ChannelConfig.PortID,
		Version:          transfertypes.V2,
		Encoding:         "json",
		Sender:           sender.GetAddress().String(),
		Receiver:         receiver.GetAddress().String(),
		TimeoutTimestamp: suite.chainA.GetTimeoutTimestamp(),
		Memo:             "",
		Tokens:           sdk.NewCoins(ibctesting.TestCoin),
		Forwarding: &typesv2.Forwarding{
			Unwind: false,
			Hops: []typesv2.Hop{
				{PortId: pathBtoC.EndpointA.ChannelConfig.PortID, ChannelId: pathBtoC.EndpointA.ChannelID},
			},
		},
	}
	result, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed
	suite.Require().NotNil(result)

	// // parse the packet from result events and recv packet on chainB
	// packetFromAtoB, err := ibctesting.ParsePacketFromEvents(result.Events)
	// suite.Require().NoError(err)
	// suite.Require().NotNil(packetFromAtoB)

	// err = pathAtoB.EndpointB.UpdateClient()
	// suite.Require().NoError(err)

	// result, err = pathAtoB.EndpointB.RecvPacketWithResult(packetFromAtoB)
	// suite.Require().NoError(err)
	// suite.Require().NotNil(result)

	// // Check that Escrow A has amount
	// suite.assertAmountOnChain(suite.chainA, escrow, amount, sdk.DefaultBondDenom)

	// // denom path: transfer/channel-0
	// denom := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	// suite.assertAmountOnChain(suite.chainB, escrow, amount, denom.IBCDenom())

	// packetFromBtoC, err := ibctesting.ParsePacketFromEvents(result.Events)
	// suite.Require().NoError(err)
	// suite.Require().NotNil(packetFromBtoC)

	// err = pathBtoC.EndpointA.UpdateClient()
	// suite.Require().NoError(err)

	// err = pathBtoC.EndpointB.UpdateClient()
	// suite.Require().NoError(err)

	// // B should have stored the forwarded packet.
	// _, found := suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoC.SourcePort, packetFromBtoC.SourceChannel, packetFromBtoC.Sequence)
	// suite.Require().True(found, "Chain B should have stored the forwarded packet")

	// result, err = pathBtoC.EndpointB.RecvPacketWithResult(packetFromBtoC)
	// suite.Require().NoError(err)
	// suite.Require().NotNil(result)

	// // transfer/channel-0/transfer/channel-0/denom
	// // Check that the final receiver has received the expected tokens.
	// denomABC := types.NewDenom(sdk.DefaultBondDenom, types.NewHop(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID), types.NewHop(pathAtoB.EndpointB.ChannelConfig.PortID, pathAtoB.EndpointB.ChannelID))
	// // Check that the final receiver has received the expected tokens.
	// suite.assertAmountOnChain(suite.chainC, balance, amount, denomABC.IBCDenom())

	// successAck := channeltypes.NewResultAcknowledgement([]byte{byte(1)})
	// successAckBz := channeltypes.CommitAcknowledgement(successAck.Acknowledgement())
	// ackOnC := suite.chainC.GetAcknowledgement(packetFromBtoC)
	// suite.Require().Equal(successAckBz, ackOnC)

	// // Ack back to B
	// err = pathBtoC.EndpointB.UpdateClient()
	// suite.Require().NoError(err)

	// err = pathBtoC.EndpointA.AcknowledgePacket(packetFromBtoC, successAck.Acknowledgement())
	// suite.Require().NoError(err)

	// ackOnB := suite.chainB.GetAcknowledgement(packetFromAtoB)
	// suite.Require().Equal(successAckBz, ackOnB)

	// // B should now have deleted the forwarded packet.
	// _, found = suite.chainB.GetSimApp().TransferKeeper.GetForwardedPacket(suite.chainB.GetContext(), packetFromBtoC.SourcePort, packetFromBtoC.SourceChannel, packetFromBtoC.Sequence)
	// suite.Require().False(found, "Chain B should have deleted the forwarded packet")

	// // Ack back to A
	// err = pathAtoB.EndpointA.UpdateClient()
	// suite.Require().NoError(err)

	// err = pathAtoB.EndpointA.AcknowledgePacket(packetFromAtoB, successAck.Acknowledgement())
	// suite.Require().NoError(err)
}
