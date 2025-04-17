package packetforward_test

import (
	"encoding/json"
	"log"
	"testing"

	"github.com/stretchr/testify/suite"

	packetforward "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware"
	packetforwardkeeper "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/keeper"
	packetforwardtypes "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	maxCallbackGas = uint64(1000000)
	V1             = "ics20-1"
)

var (
	testDenom  = "uatom"
	testAmount = "100"
)

// CallbacksTestSuite defines the needed instances and methods to test callbacks
type PFMTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain

	pathAB *ibctesting.Path
	pathBC *ibctesting.Path
}

func TestPFMTestSuite(t *testing.T) {
	suite.Run(t, new(PFMTestSuite))
}

// setupChains sets up a coordinator with 2 test chains.
func (s *PFMTestSuite) setupChains() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))

	s.pathAB = ibctesting.NewTransferPath(s.chainA, s.chainB)
	s.pathAB.Setup()

	s.pathBC = ibctesting.NewTransferPath(s.chainB, s.chainC)
	s.pathBC.Setup()
}

func (s *PFMTestSuite) TestOnRecvPacket() {
	s.T().Skip("Later")
	s.setupChains()

	pfmKeeper := s.chainA.GetSimApp().PFMKeeper

	ibcModule, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
	s.Require().True(ok)

	ibcMiddleware := packetforward.NewIBCMiddleware(ibcModule, &pfmKeeper, 0, packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp)
	ctx := s.chainA.GetContext()
	version := s.pathAB.EndpointA.GetChannel().Version
	relayerAddr := s.chainA.SenderAccount.GetAddress()
	ack := ibcMiddleware.OnRecvPacket(ctx, version, channeltypes.Packet{}, relayerAddr)
	s.Require().True(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}

	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)

	s.Require().Equal("", expectedAck.GetError())

}

func (s *PFMTestSuite) TestOnRecvPacket_Nomemo() {
	s.setupChains()

	pfmKeeper := s.chainA.GetSimApp().PFMKeeper

	ibcModule, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
	s.Require().True(ok)

	ibcMiddleware := packetforward.NewIBCMiddleware(ibcModule, &pfmKeeper, 0, packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp)
	ctx := s.chainA.GetContext()
	version := s.pathAB.EndpointA.GetChannel().Version
	relayerAddr := s.chainA.SenderAccount.GetAddress()
	receiverAddr := s.chainB.SenderAccount.GetAddress()

	packet := s.transferPacket(relayerAddr.String(), receiverAddr.String(), s.pathAB, 0, "{}")

	ack := ibcMiddleware.OnRecvPacket(ctx, version, packet, relayerAddr)
	s.Require().True(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}

	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)

	s.Require().Equal("", expectedAck.GetError())
	s.Require().ElementsMatch([]byte{1}, expectedAck.GetResult())
}

func (s *PFMTestSuite) TestOnRecvPacket_InvalidReceiver() {
	s.setupChains()

	pfmKeeper := s.chainA.GetSimApp().PFMKeeper

	ibcModule, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
	s.Require().True(ok)

	ibcMiddleware := packetforward.NewIBCMiddleware(ibcModule, &pfmKeeper, 0, packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp)
	ctx := s.chainA.GetContext()
	version := s.pathAB.EndpointA.GetChannel().Version
	relayerAddr := s.chainA.SenderAccount.GetAddress()

	packet := s.transferPacket(relayerAddr.String(), "", s.pathAB, 0, nil)

	ack := ibcMiddleware.OnRecvPacket(ctx, version, packet, relayerAddr)
	s.Require().False(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}

	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)

	s.Require().Equal("ABCI code: 5: error handling packet: see events for details", expectedAck.GetError())
	s.Require().Empty(expectedAck.GetResult())
}

func (s *PFMTestSuite) TestOnRecvPacket_NoForward() {
	s.setupChains()

	pfmKeeper := s.chainA.GetSimApp().PFMKeeper

	ibcModule, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
	s.Require().True(ok)

	ibcMiddleware := packetforward.NewIBCMiddleware(ibcModule, &pfmKeeper, 0, packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp)
	ctx := s.chainA.GetContext()
	version := s.pathAB.EndpointA.GetChannel().Version

	senderAddr := s.chainA.SenderAccount.GetAddress()
	receiverAddr := s.chainB.SenderAccount.GetAddress()

	packet := s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathAB, 0, nil)

	ack := ibcMiddleware.OnRecvPacket(ctx, version, packet, senderAddr)
	s.Require().True(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}

	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)
	s.Require().Equal("", expectedAck.GetError())

	s.Require().Equal([]byte{1}, expectedAck.GetResult())
}

func (s *PFMTestSuite) TestOnRecvPacket_RecvPacketFailed() {
	s.setupChains()
	ctx := s.chainA.GetContext()

	pfmKeeper := s.chainA.GetSimApp().PFMKeeper
	transferKeeper := s.chainA.GetSimApp().TransferKeeper

	// Also can be done if send amount is 0
	transferKeeper.SetParams(ctx, transfertypes.Params{ReceiveEnabled: false})

	ibcModule, ok := s.chainA.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
	s.Require().True(ok)

	ibcMiddleware := packetforward.NewIBCMiddleware(ibcModule, &pfmKeeper, 0, packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp)
	version := s.pathAB.EndpointA.GetChannel().Version

	senderAddr := s.chainA.SenderAccount.GetAddress()
	receiverAddr := s.chainB.SenderAccount.GetAddress()
	metadata := &packetforwardtypes.PacketMetadata{
		Forward: &packetforwardtypes.ForwardMetadata{
			Receiver: receiverAddr.String(),
			Port:     s.pathAB.EndpointA.ChannelConfig.PortID,
			Channel:  s.pathAB.EndpointA.ChannelID,
		},
	}
	packet := s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathAB, 0, metadata)

	ack := ibcMiddleware.OnRecvPacket(ctx, version, packet, senderAddr)
	s.Require().False(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}

	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)
	log.Printf("Error: %s", expectedAck.GetError())
	s.Require().Equal("packet-forward-middleware error: error receiving packet: ack error: {\"error\":\"ABCI code: 8: error handling packet: see events for details\"}", expectedAck.GetError())

	s.Require().Equal([]byte(nil), expectedAck.GetResult())
}

func (s *PFMTestSuite) TestOnRecvPacket_ForwardNoFee() {
	s.setupChains()
	ctxB := s.chainB.GetContext()

	pfmB := s.ibcMiddleware(s.chainB, transfertypes.ModuleName)
	senderAddr := s.chainA.SenderAccount.GetAddress()
	receiverAddr := s.chainC.SenderAccount.GetAddress()

	metadata := &packetforwardtypes.PacketMetadata{
		Forward: &packetforwardtypes.ForwardMetadata{
			Receiver: receiverAddr.String(),
			Port:     s.pathBC.EndpointA.ChannelConfig.PortID,
			Channel:  s.pathBC.EndpointA.ChannelID,
		},
	}
	packet := s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathAB, 0, metadata)

	version := s.pathAB.EndpointA.GetChannel().Version
	ack := pfmB.OnRecvPacket(ctxB, version, packet, senderAddr)
	s.Require().Nil(ack)

	// Check that chain C has received the packet
	//
	ctxC := s.chainC.GetContext()
	pfmC := s.ibcMiddleware(s.chainC, transfertypes.ModuleName)

	packet = s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathBC, 0, nil)

	version = s.pathBC.EndpointA.GetChannel().Version
	ack = pfmC.OnRecvPacket(ctxC, version, packet, senderAddr)

	// Ack on chainC
	packet = s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathBC, 1, nil)
	err := pfmC.OnAcknowledgementPacket(ctxC, version, packet, ack.Acknowledgement(), senderAddr)
	s.Require().NoError(err)

	// Ack on ChainB
	err = pfmB.OnAcknowledgementPacket(ctxB, version, packet, ack.Acknowledgement(), senderAddr)
	s.Require().NoError(err)
}

func (s *PFMTestSuite) ibcMiddleware(chain *ibctesting.TestChain, module string) packetforward.IBCMiddleware {
	pfmKeeper := chain.GetSimApp().PFMKeeper

	ibcModule, ok := chain.App.GetIBCKeeper().PortKeeper.Route(module)
	s.Require().True(ok)

	ibcMiddleware := packetforward.NewIBCMiddleware(ibcModule, &pfmKeeper, 0, packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp)
	return ibcMiddleware
}

func (s *PFMTestSuite) transferPacket(sender string, receiver string, path *ibctesting.Path, seq uint64, metadata any) channeltypes.Packet {
	s.T().Helper()
	tokenPacket := transfertypes.FungibleTokenPacketData{
		Denom:    testDenom,
		Amount:   testAmount,
		Sender:   sender,
		Receiver: receiver,
	}

	if metadata != nil {
		if mStr, ok := metadata.(string); ok {
			tokenPacket.Memo = mStr
		} else {
			memo, err := json.Marshal(metadata)
			s.Require().NoError(err)
			tokenPacket.Memo = string(memo)
		}
	}

	tokenData, err := transfertypes.ModuleCdc.MarshalJSON(&tokenPacket)
	s.Require().NoError(err)

	return channeltypes.Packet{
		SourcePort:         path.EndpointA.ChannelConfig.PortID,
		SourceChannel:      path.EndpointA.ChannelID,
		DestinationPort:    path.EndpointB.ChannelConfig.PortID,
		DestinationChannel: path.EndpointB.ChannelID,
		Data:               tokenData,
		Sequence:           seq,
	}
}
