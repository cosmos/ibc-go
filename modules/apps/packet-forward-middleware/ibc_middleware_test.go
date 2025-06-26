package packetforward_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	packetforward "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware"
	packetforwardkeeper "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/keeper"
	packetforwardtypes "github.com/cosmos/ibc-go/v10/modules/apps/packet-forward-middleware/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	ibcmock "github.com/cosmos/ibc-go/v10/testing/mock"
)

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

// setupChains sets up a coordinator with 3 test chains.
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

func (s *PFMTestSuite) TestSetICS4Wrapper() {
	s.setupChains()

	pfm := s.pktForwardMiddleware(s.chainA)

	s.Require().Panics(func() {
		pfm.SetICS4Wrapper(nil)
	}, "ICS4Wrapper cannot be nil")

	s.Require().NotPanics(func() {
		pfm.SetICS4Wrapper(s.chainA.App.GetIBCKeeper().ChannelKeeper)
	}, "ICS4Wrapper should be set without panic")
}

func (s *PFMTestSuite) TestSetUnderlyingApplication() {
	s.setupChains()

	pfmKeeper := s.chainA.GetSimApp().PFMKeeper

	pfm := packetforward.NewIBCMiddleware(pfmKeeper, 0, packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp)

	s.Require().Panics(func() {
		pfm.SetUnderlyingApplication(nil)
	}, "underlying application cannot be nil")

	s.Require().NotPanics(func() {
		pfm.SetUnderlyingApplication(&ibcmock.IBCModule{})
	}, "underlying application should be set without panic")

	s.Require().Panics(func() {
		pfm.SetUnderlyingApplication(&ibcmock.IBCModule{})
	}, "underlying application should not be set again")
}

func (s *PFMTestSuite) TestOnRecvPacket_NonfungibleToken() {
	s.setupChains()

	ctx := s.chainA.GetContext()
	version := s.pathAB.EndpointA.GetChannel().Version
	relayerAddr := s.chainA.SenderAccount.GetAddress()

	pfm := s.pktForwardMiddleware(s.chainA)
	ack := pfm.OnRecvPacket(ctx, version, channeltypes.Packet{}, relayerAddr)
	s.Require().False(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}
	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)

	// Transfer keeper returns this error if the packet received is not a fungible token.
	s.Require().Equal("ABCI code: 12: error handling packet: see events for details", expectedAck.GetError())
}

func (s *PFMTestSuite) TestOnRecvPacket_NoMemo() {
	s.setupChains()

	ctx := s.chainA.GetContext()
	version := s.pathAB.EndpointA.GetChannel().Version
	relayerAddr := s.chainA.SenderAccount.GetAddress()
	receiverAddr := s.chainB.SenderAccount.GetAddress()

	packet := s.transferPacket(relayerAddr.String(), receiverAddr.String(), s.pathAB, 0, "{}")

	pfm := s.pktForwardMiddleware(s.chainA)
	ack := pfm.OnRecvPacket(ctx, version, packet, relayerAddr)
	s.Require().True(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}
	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)

	s.Require().Empty(expectedAck.GetError())
	s.Require().ElementsMatch([]byte{1}, expectedAck.GetResult())
}

func (s *PFMTestSuite) TestOnRecvPacket_InvalidReceiver() {
	s.setupChains()

	ctx := s.chainA.GetContext()
	version := s.pathAB.EndpointA.GetChannel().Version
	relayerAddr := s.chainA.SenderAccount.GetAddress()

	packet := s.transferPacket(relayerAddr.String(), "", s.pathAB, 0, "")

	pfm := s.pktForwardMiddleware(s.chainA)
	ack := pfm.OnRecvPacket(ctx, version, packet, relayerAddr)
	s.Require().False(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}
	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)

	s.Require().Equal("ABCI code: 5: error handling packet: see events for details", expectedAck.GetError())
	s.Require().Empty(expectedAck.GetResult())
}

func (s *PFMTestSuite) TestOnRecvPacket_NoForward() {
	s.setupChains()

	ctx := s.chainA.GetContext()
	version := s.pathAB.EndpointA.GetChannel().Version

	senderAddr := s.chainA.SenderAccount.GetAddress()
	receiverAddr := s.chainB.SenderAccount.GetAddress()

	packet := s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathAB, 0, "")

	pfm := s.pktForwardMiddleware(s.chainA)
	ack := pfm.OnRecvPacket(ctx, version, packet, senderAddr)
	s.Require().True(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}
	err := s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)
	s.Require().Empty(expectedAck.GetError())

	s.Require().Equal([]byte{1}, expectedAck.GetResult())
}

func (s *PFMTestSuite) TestOnRecvPacket_RecvPacketFailed() {
	s.setupChains()

	transferKeeper := s.chainA.GetSimApp().TransferKeeper
	ctx := s.chainA.GetContext()
	transferKeeper.SetParams(ctx, transfertypes.Params{ReceiveEnabled: false})

	version := s.pathAB.EndpointA.GetChannel().Version

	senderAddr := s.chainA.SenderAccount.GetAddress()
	receiverAddr := s.chainB.SenderAccount.GetAddress()
	metadata := &packetforwardtypes.PacketMetadata{
		Forward: packetforwardtypes.ForwardMetadata{
			Receiver: receiverAddr.String(),
			Port:     s.pathAB.EndpointA.ChannelConfig.PortID,
			Channel:  s.pathAB.EndpointA.ChannelID,
		},
	}
	metadataJSON, err := metadata.ToMemo()
	s.Require().NoError(err)
	packet := s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathAB, 0, metadataJSON)

	pfm := s.pktForwardMiddleware(s.chainA)
	ack := pfm.OnRecvPacket(ctx, version, packet, senderAddr)
	s.Require().False(ack.Success())

	expectedAck := &channeltypes.Acknowledgement{}

	err = s.chainA.Codec.UnmarshalJSON(ack.Acknowledgement(), expectedAck)
	s.Require().NoError(err)
	s.Require().Equal("packet-forward-middleware error: error receiving packet: ack error: {\"error\":\"ABCI code: 8: error handling packet: see events for details\"}", expectedAck.GetError())

	s.Require().Equal([]byte(nil), expectedAck.GetResult())
}

func (s *PFMTestSuite) TestOnRecvPacket_ForwardNoFee() {
	s.setupChains()

	senderAddr := s.chainA.SenderAccount.GetAddress()
	receiverAddr := s.chainC.SenderAccount.GetAddress()
	metadata := &packetforwardtypes.PacketMetadata{
		Forward: packetforwardtypes.ForwardMetadata{
			Receiver: receiverAddr.String(),
			Port:     s.pathBC.EndpointA.ChannelConfig.PortID,
			Channel:  s.pathBC.EndpointA.ChannelID,
		},
	}
	metadataJSON, err := metadata.ToMemo()
	s.Require().NoError(err)
	packet := s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathAB, 0, metadataJSON)
	version := s.pathAB.EndpointA.GetChannel().Version
	ctxB := s.chainB.GetContext()

	pfmB := s.pktForwardMiddleware(s.chainB)
	ack := pfmB.OnRecvPacket(ctxB, version, packet, senderAddr)
	s.Require().Nil(ack)

	// Check that chain C has received the packet
	ctxC := s.chainC.GetContext()
	packet = s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathBC, 0, "")
	version = s.pathBC.EndpointA.GetChannel().Version

	pfmC := s.pktForwardMiddleware(s.chainC)
	ack = pfmC.OnRecvPacket(ctxC, version, packet, senderAddr)
	s.Require().NotNil(ack)

	// Ack on chainC
	packet = s.transferPacket(senderAddr.String(), receiverAddr.String(), s.pathBC, 1, "")
	err = pfmC.OnAcknowledgementPacket(ctxC, version, packet, ack.Acknowledgement(), senderAddr)
	s.Require().NoError(err)

	// Ack on ChainB
	err = pfmB.OnAcknowledgementPacket(ctxB, version, packet, ack.Acknowledgement(), senderAddr)
	s.Require().NoError(err)
}

func (s *PFMTestSuite) pktForwardMiddleware(chain *ibctesting.TestChain) *packetforward.IBCMiddleware {
	pfmKeeper := chain.GetSimApp().PFMKeeper

	ibcModule, ok := chain.App.GetIBCKeeper().PortKeeper.Route(transfertypes.ModuleName)
	s.Require().True(ok)

	transferStack, ok := ibcModule.(porttypes.PacketUnmarshalerModule)
	s.Require().True(ok)

	ibcMiddleware := packetforward.NewIBCMiddleware(pfmKeeper, 0, packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp)
	ibcMiddleware.SetUnderlyingApplication(transferStack)
	return ibcMiddleware
}

func (s *PFMTestSuite) transferPacket(sender string, receiver string, path *ibctesting.Path, seq uint64, metadata string) channeltypes.Packet {
	s.T().Helper()
	tokenPacket := transfertypes.FungibleTokenPacketData{
		Denom:    "uatom",
		Amount:   "100",
		Sender:   sender,
		Receiver: receiver,
		Memo:     metadata,
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
