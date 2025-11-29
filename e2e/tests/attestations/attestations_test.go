//go:build !test_e2e

package attestations

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	test "github.com/cosmos/interchaintest/v10/testutil"
	"github.com/ethereum/go-ethereum/crypto"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	attestations "github.com/cosmos/ibc-go/v10/modules/light-clients/10-attestations"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestAttestationsTestSuite(t *testing.T) {
	testifysuite.Run(t, new(AttestationsTestSuite))
}

type AttestationsTestSuite struct {
	testsuite.E2ETestSuite
	attestorKeys []*ecdsa.PrivateKey
}

func (s *AttestationsTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 1, nil)
	s.setupAttestors()
}

func (s *AttestationsTestSuite) setupAttestors() {
	for range 3 {
		privateKey, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.attestorKeys = append(s.attestorKeys, privateKey)
	}
}

func (s *AttestationsTestSuite) getAttestorAddresses() []string {
	var addresses []string
	for _, key := range s.attestorKeys {
		address := crypto.PubkeyToAddress(key.PublicKey)
		addresses = append(addresses, address.Hex())
	}
	return addresses
}

func (s *AttestationsTestSuite) signAttestationData(data []byte) [][]byte {
	hash := sha256.Sum256(data)
	var signatures [][]byte
	for _, key := range s.attestorKeys {
		sig, err := crypto.Sign(hash[:], key)
		s.Require().NoError(err)
		signatures = append(signatures, sig)
	}
	return signatures
}

func (s *AttestationsTestSuite) createPacketAttestationProof(height uint64, path []byte, commitment []byte) []byte {
	packetAttestation := &attestations.PacketAttestation{
		Height: height,
		Packets: []attestations.PacketCompact{
			{
				Path:       path,
				Commitment: commitment,
			},
		},
	}
	attestationData, err := proto.Marshal(packetAttestation)
	s.Require().NoError(err)

	signatures := s.signAttestationData(attestationData)
	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      signatures,
	}
	proofBz, err := proto.Marshal(proof)
	s.Require().NoError(err)
	return proofBz
}

func (s *AttestationsTestSuite) createNonMembershipProof(height uint64, path []byte) []byte {
	zeroCommitment := make([]byte, 32)
	packetAttestation := &attestations.PacketAttestation{
		Height: height,
		Packets: []attestations.PacketCompact{
			{
				Path:       path,
				Commitment: zeroCommitment,
			},
		},
	}
	attestationData, err := proto.Marshal(packetAttestation)
	s.Require().NoError(err)

	signatures := s.signAttestationData(attestationData)
	proof := &attestations.AttestationProof{
		AttestationData: attestationData,
		Signatures:      signatures,
	}
	proofBz, err := proto.Marshal(proof)
	s.Require().NoError(err)
	return proofBz
}

func (*AttestationsTestSuite) hashPath(path []byte) []byte {
	hash := sha256.Sum256(path)
	return hash[:]
}

func (s *AttestationsTestSuite) prefixedPath(key []byte) []byte {
	prefixedPath := bytes.Join([][]byte{[]byte(ibcexported.StoreKey), key}, []byte("/"))
	return s.hashPath(prefixedPath)
}

func (s *AttestationsTestSuite) TestMsgTransfer_Attestations() {
	t := s.T()
	ctx := context.TODO()

	chains := s.GetAllChains()
	chainA := chains[0]
	chainADenom := chainA.Config().Denom
	cfg := chainA.Config().EncodingConfig

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		clientIDA          string
		clientIDB          string
		connectionIDA      string
		connectionIDB      string
		channelIDA         string
		channelIDB         string
		packet             channeltypes.Packet
		ack                []byte
		proofHeight        uint64 = 100
		proofTimestamp     uint64
		msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
		msgChanOpenTryRes  channeltypes.MsgChannelOpenTryResponse
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")

	proofTimestamp = uint64(time.Now().UnixNano())

	t.Run("create attestations clients", func(t *testing.T) {
		attestorAddresses := s.getAttestorAddresses()

		clientStateA := attestations.NewClientState(attestorAddresses, 2, proofHeight)
		consensusStateA := &attestations.ConsensusState{Timestamp: proofTimestamp}

		msgCreateClientA, err := clienttypes.NewMsgCreateClient(clientStateA, consensusStateA, rlyWallet.FormattedAddress())
		s.Require().NoError(err)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgCreateClientA)
		s.AssertTxSuccess(txResp)

		var createClientRes clienttypes.MsgCreateClientResponse
		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &createClientRes))
		clientIDA = createClientRes.ClientId

		clientStateB := attestations.NewClientState(attestorAddresses, 2, proofHeight)
		consensusStateB := &attestations.ConsensusState{Timestamp: proofTimestamp}

		msgCreateClientB, err := clienttypes.NewMsgCreateClient(clientStateB, consensusStateB, rlyWallet.FormattedAddress())
		s.Require().NoError(err)

		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgCreateClientB)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &createClientRes))
		clientIDB = createClientRes.ClientId

		t.Logf("Created clients: %s, %s", clientIDA, clientIDB)
	})

	t.Run("verify attestations client status is active", func(t *testing.T) {
		status, err := query.ClientStatus(ctx, chainA, clientIDA)
		s.Require().NoError(err)
		s.Require().Equal(ibcexported.Active.String(), status)

		status, err = query.ClientStatus(ctx, chainA, clientIDB)
		s.Require().NoError(err)
		s.Require().Equal(ibcexported.Active.String(), status)
	})

	t.Run("connection open init", func(t *testing.T) {
		msgConnOpenInit := connectiontypes.NewMsgConnectionOpenInit(
			clientIDA,
			clientIDB,
			commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey)),
			ibctesting.ConnectionVersion,
			0,
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgConnOpenInit)
		s.AssertTxSuccess(txResp)

		var err error
		connectionIDA, err = ibctesting.ParseConnectionIDFromEvents(txResp.Events)
		s.Require().NoError(err)
		t.Logf("Connection init: %s", connectionIDA)
	})

	t.Run("connection open try", func(t *testing.T) {
		connPath := s.prefixedPath(host.ConnectionKey(connectionIDA))

		connEnd := connectiontypes.NewConnectionEnd(
			connectiontypes.INIT,
			clientIDA,
			connectiontypes.NewCounterparty(clientIDB, "", commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey))),
			[]*connectiontypes.Version{ibctesting.ConnectionVersion},
			0,
		)
		connEndBz, err := cfg.Codec.Marshal(&connEnd)
		s.Require().NoError(err)
		connHash := sha256.Sum256(connEndBz)

		proofInit := s.createPacketAttestationProof(proofHeight, connPath, connHash[:])

		msgConnOpenTry := connectiontypes.NewMsgConnectionOpenTry(
			clientIDB,
			connectionIDA,
			clientIDA,
			commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey)),
			[]*connectiontypes.Version{ibctesting.ConnectionVersion},
			0,
			proofInit,
			clienttypes.NewHeight(0, proofHeight),
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgConnOpenTry)
		s.AssertTxSuccess(txResp)

		connectionIDB, err = ibctesting.ParseConnectionIDFromEvents(txResp.Events)
		s.Require().NoError(err)
		t.Logf("Connection try: %s", connectionIDB)
	})

	t.Run("connection open ack", func(t *testing.T) {
		connPath := s.prefixedPath(host.ConnectionKey(connectionIDB))

		connEnd := connectiontypes.NewConnectionEnd(
			connectiontypes.TRYOPEN,
			clientIDB,
			connectiontypes.NewCounterparty(clientIDA, connectionIDA, commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey))),
			[]*connectiontypes.Version{ibctesting.ConnectionVersion},
			0,
		)
		connEndBz, err := cfg.Codec.Marshal(&connEnd)
		s.Require().NoError(err)
		connHash := sha256.Sum256(connEndBz)

		proofTry := s.createPacketAttestationProof(proofHeight, connPath, connHash[:])

		msgConnOpenAck := connectiontypes.NewMsgConnectionOpenAck(
			connectionIDA,
			connectionIDB,
			proofTry,
			clienttypes.NewHeight(0, proofHeight),
			ibctesting.ConnectionVersion,
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgConnOpenAck)
		s.AssertTxSuccess(txResp)
		t.Log("Connection ack completed")
	})

	t.Run("connection open confirm", func(t *testing.T) {
		connPath := s.prefixedPath(host.ConnectionKey(connectionIDA))

		connEnd := connectiontypes.NewConnectionEnd(
			connectiontypes.OPEN,
			clientIDA,
			connectiontypes.NewCounterparty(clientIDB, connectionIDB, commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey))),
			[]*connectiontypes.Version{ibctesting.ConnectionVersion},
			0,
		)
		connEndBz, err := cfg.Codec.Marshal(&connEnd)
		s.Require().NoError(err)
		connHash := sha256.Sum256(connEndBz)

		proofAck := s.createPacketAttestationProof(proofHeight, connPath, connHash[:])

		msgConnOpenConfirm := connectiontypes.NewMsgConnectionOpenConfirm(
			connectionIDB,
			proofAck,
			clienttypes.NewHeight(0, proofHeight),
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgConnOpenConfirm)
		s.AssertTxSuccess(txResp)
		t.Log("Connection confirm completed")
	})

	channelVersion := transfertypes.V1

	t.Run("channel open init", func(t *testing.T) {
		msgChanOpenInit := channeltypes.NewMsgChannelOpenInit(
			transfertypes.PortID, channelVersion,
			channeltypes.UNORDERED, []string{connectionIDA},
			transfertypes.PortID, rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenInit)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))
		channelIDA = msgChanOpenInitRes.ChannelId
		t.Logf("Channel init: %s", channelIDA)
	})

	t.Run("channel open try", func(t *testing.T) {
		chanPath := s.prefixedPath(host.ChannelKey(transfertypes.PortID, channelIDA))

		chanEnd := channeltypes.NewChannel(
			channeltypes.INIT,
			channeltypes.UNORDERED,
			channeltypes.NewCounterparty(transfertypes.PortID, ""),
			[]string{connectionIDA},
			channelVersion,
		)
		chanEndBz, err := cfg.Codec.Marshal(&chanEnd)
		s.Require().NoError(err)
		chanHash := sha256.Sum256(chanEndBz)

		proofInit := s.createPacketAttestationProof(proofHeight, chanPath, chanHash[:])

		msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
			transfertypes.PortID, channelVersion,
			channeltypes.UNORDERED, []string{connectionIDB},
			transfertypes.PortID, channelIDA,
			channelVersion, proofInit, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenTry)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))
		channelIDB = msgChanOpenTryRes.ChannelId
		t.Logf("Channel try: %s", channelIDB)
	})

	t.Run("channel open ack", func(t *testing.T) {
		chanPath := s.prefixedPath(host.ChannelKey(transfertypes.PortID, channelIDB))

		chanEnd := channeltypes.NewChannel(
			channeltypes.TRYOPEN,
			channeltypes.UNORDERED,
			channeltypes.NewCounterparty(transfertypes.PortID, channelIDA),
			[]string{connectionIDB},
			channelVersion,
		)
		chanEndBz, err := cfg.Codec.Marshal(&chanEnd)
		s.Require().NoError(err)
		chanHash := sha256.Sum256(chanEndBz)

		proofTry := s.createPacketAttestationProof(proofHeight, chanPath, chanHash[:])

		msgChanOpenAck := channeltypes.NewMsgChannelOpenAck(
			transfertypes.PortID, channelIDA,
			channelIDB, channelVersion,
			proofTry, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenAck)
		s.AssertTxSuccess(txResp)
		t.Log("Channel ack completed")
	})

	t.Run("channel open confirm", func(t *testing.T) {
		chanPath := s.prefixedPath(host.ChannelKey(transfertypes.PortID, channelIDA))

		chanEnd := channeltypes.NewChannel(
			channeltypes.OPEN,
			channeltypes.UNORDERED,
			channeltypes.NewCounterparty(transfertypes.PortID, channelIDB),
			[]string{connectionIDA},
			channelVersion,
		)
		chanEndBz, err := cfg.Codec.Marshal(&chanEnd)
		s.Require().NoError(err)
		chanHash := sha256.Sum256(chanEndBz)

		proofAck := s.createPacketAttestationProof(proofHeight, chanPath, chanHash[:])

		msgChanOpenConfirm := channeltypes.NewMsgChannelOpenConfirm(
			transfertypes.PortID, channelIDB,
			proofAck, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenConfirm)
		s.AssertTxSuccess(txResp)
		t.Log("Channel confirm completed")
	})

	t.Run("query channels", func(t *testing.T) {
		channelEndA, err := query.Channel(ctx, chainA, transfertypes.PortID, channelIDA)
		s.Require().NoError(err)
		s.Require().Equal(channeltypes.OPEN, channelEndA.State)

		channelEndB, err := query.Channel(ctx, chainA, transfertypes.PortID, channelIDB)
		s.Require().NoError(err)
		s.Require().Equal(channeltypes.OPEN, channelEndB.State)
	})

	t.Run("send IBC transfer", func(t *testing.T) {
		txResp := s.Transfer(ctx, chainA, userAWallet, transfertypes.PortID, channelIDA, testvalues.DefaultTransferAmount(chainADenom), userAWallet.FormattedAddress(), userBWallet.FormattedAddress(), clienttypes.NewHeight(1, 500), 0, "")
		s.AssertTxSuccess(txResp)

		var err error
		packet, err = ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
		t.Logf("Packet sent: seq=%d", packet.Sequence)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("recv packet with attestation proof", func(t *testing.T) {
		packetCommitment := channeltypes.CommitPacket(packet)
		packetPath := s.prefixedPath(host.PacketCommitmentKey(packet.SourcePort, packet.SourceChannel, packet.Sequence))

		proofCommitment := s.createPacketAttestationProof(proofHeight, packetPath, packetCommitment)

		msgRecvPacket := channeltypes.NewMsgRecvPacket(packet, proofCommitment, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		var err error
		ack, err = ibctesting.ParseAckFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ack)
		t.Log("Packet received")
	})

	t.Run("acknowledge packet with attestation proof", func(t *testing.T) {
		ackCommitment := channeltypes.CommitAcknowledgement(ack)
		ackPath := s.prefixedPath(host.PacketAcknowledgementKey(packet.DestinationPort, packet.DestinationChannel, packet.Sequence))

		proofAcked := s.createPacketAttestationProof(proofHeight, ackPath, ackCommitment)

		msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, proofAcked, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
		s.AssertTxSuccess(txResp)
		t.Log("Packet acknowledged")
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, transfertypes.PortID, channelIDA, 1)

		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, channelIDB)
		actualBalance, err := query.Balance(ctx, chainA, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
		t.Logf("User B received %d of %s", actualBalance.Int64(), ibcToken.IBCDenom())
	})

	t.Run("send IBC transfer back (unwind)", func(t *testing.T) {
		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, channelIDB)

		txResp := s.Transfer(ctx, chainA, userBWallet, transfertypes.PortID, channelIDB, testvalues.DefaultTransferAmount(ibcToken.IBCDenom()), userBWallet.FormattedAddress(), userAWallet.FormattedAddress(), clienttypes.NewHeight(1, 500), 0, "")
		s.AssertTxSuccess(txResp)

		var err error
		packet, err = ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
		t.Logf("Return packet sent: seq=%d", packet.Sequence)
	})

	t.Run("recv return packet", func(t *testing.T) {
		packetCommitment := channeltypes.CommitPacket(packet)
		packetPath := s.prefixedPath(host.PacketCommitmentKey(packet.SourcePort, packet.SourceChannel, packet.Sequence))

		proofCommitment := s.createPacketAttestationProof(proofHeight, packetPath, packetCommitment)

		msgRecvPacket := channeltypes.NewMsgRecvPacket(packet, proofCommitment, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		var err error
		ack, err = ibctesting.ParseAckFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ack)
		t.Log("Return packet received")
	})

	t.Run("acknowledge return packet", func(t *testing.T) {
		ackCommitment := channeltypes.CommitAcknowledgement(ack)
		ackPath := s.prefixedPath(host.PacketAcknowledgementKey(packet.DestinationPort, packet.DestinationChannel, packet.Sequence))

		proofAcked := s.createPacketAttestationProof(proofHeight, ackPath, ackCommitment)

		msgAcknowledgement := channeltypes.NewMsgAcknowledgement(packet, ack, proofAcked, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
		s.AssertTxSuccess(txResp)
		t.Log("Return packet acknowledged")
	})

	t.Run("verify tokens unwound", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, transfertypes.PortID, channelIDB, 1)

		actualBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
		t.Logf("User A recovered full balance: %d", actualBalance)

		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, channelIDB)
		userBBalance, err := query.Balance(ctx, chainA, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(userBBalance.Int64())
		t.Log("User B IBC token balance is zero")
	})
}

func (s *AttestationsTestSuite) TestMsgTransfer_Timeout_Attestations() {
	t := s.T()
	ctx := context.TODO()

	chains := s.GetAllChains()
	chainA := chains[0]
	chainADenom := chainA.Config().Denom
	cfg := chainA.Config().EncodingConfig

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		clientIDA          string
		clientIDB          string
		connectionIDA      string
		connectionIDB      string
		channelIDA         string
		channelIDB         string
		packet             channeltypes.Packet
		proofHeight        uint64 = 100
		proofTimestamp     uint64
		msgChanOpenInitRes channeltypes.MsgChannelOpenInitResponse
		msgChanOpenTryRes  channeltypes.MsgChannelOpenTryResponse
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA), "failed to wait for blocks")

	proofTimestamp = uint64(time.Now().UnixNano())

	t.Run("create attestations clients", func(t *testing.T) {
		attestorAddresses := s.getAttestorAddresses()

		clientStateA := attestations.NewClientState(attestorAddresses, 2, proofHeight)
		consensusStateA := &attestations.ConsensusState{Timestamp: proofTimestamp}

		msgCreateClientA, err := clienttypes.NewMsgCreateClient(clientStateA, consensusStateA, rlyWallet.FormattedAddress())
		s.Require().NoError(err)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgCreateClientA)
		s.AssertTxSuccess(txResp)

		var createClientRes clienttypes.MsgCreateClientResponse
		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &createClientRes))
		clientIDA = createClientRes.ClientId

		clientStateB := attestations.NewClientState(attestorAddresses, 2, proofHeight)
		consensusStateB := &attestations.ConsensusState{Timestamp: proofTimestamp}

		msgCreateClientB, err := clienttypes.NewMsgCreateClient(clientStateB, consensusStateB, rlyWallet.FormattedAddress())
		s.Require().NoError(err)

		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgCreateClientB)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &createClientRes))
		clientIDB = createClientRes.ClientId
	})

	t.Run("connection open init", func(t *testing.T) {
		msgConnOpenInit := connectiontypes.NewMsgConnectionOpenInit(
			clientIDA,
			clientIDB,
			commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey)),
			ibctesting.ConnectionVersion,
			0,
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgConnOpenInit)
		s.AssertTxSuccess(txResp)

		var err error
		connectionIDA, err = ibctesting.ParseConnectionIDFromEvents(txResp.Events)
		s.Require().NoError(err)
	})

	t.Run("connection open try", func(t *testing.T) {
		connPath := s.prefixedPath(host.ConnectionKey(connectionIDA))

		connEnd := connectiontypes.NewConnectionEnd(
			connectiontypes.INIT,
			clientIDA,
			connectiontypes.NewCounterparty(clientIDB, "", commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey))),
			[]*connectiontypes.Version{ibctesting.ConnectionVersion},
			0,
		)
		connEndBz, err := cfg.Codec.Marshal(&connEnd)
		s.Require().NoError(err)
		connHash := sha256.Sum256(connEndBz)

		proofInit := s.createPacketAttestationProof(proofHeight, connPath, connHash[:])

		msgConnOpenTry := connectiontypes.NewMsgConnectionOpenTry(
			clientIDB,
			connectionIDA,
			clientIDA,
			commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey)),
			[]*connectiontypes.Version{ibctesting.ConnectionVersion},
			0,
			proofInit,
			clienttypes.NewHeight(0, proofHeight),
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgConnOpenTry)
		s.AssertTxSuccess(txResp)

		connectionIDB, err = ibctesting.ParseConnectionIDFromEvents(txResp.Events)
		s.Require().NoError(err)
	})

	t.Run("connection open ack", func(t *testing.T) {
		connPath := s.prefixedPath(host.ConnectionKey(connectionIDB))

		connEnd := connectiontypes.NewConnectionEnd(
			connectiontypes.TRYOPEN,
			clientIDB,
			connectiontypes.NewCounterparty(clientIDA, connectionIDA, commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey))),
			[]*connectiontypes.Version{ibctesting.ConnectionVersion},
			0,
		)
		connEndBz, err := cfg.Codec.Marshal(&connEnd)
		s.Require().NoError(err)
		connHash := sha256.Sum256(connEndBz)

		proofTry := s.createPacketAttestationProof(proofHeight, connPath, connHash[:])

		msgConnOpenAck := connectiontypes.NewMsgConnectionOpenAck(
			connectionIDA,
			connectionIDB,
			proofTry,
			clienttypes.NewHeight(0, proofHeight),
			ibctesting.ConnectionVersion,
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgConnOpenAck)
		s.AssertTxSuccess(txResp)
	})

	t.Run("connection open confirm", func(t *testing.T) {
		connPath := s.prefixedPath(host.ConnectionKey(connectionIDA))

		connEnd := connectiontypes.NewConnectionEnd(
			connectiontypes.OPEN,
			clientIDA,
			connectiontypes.NewCounterparty(clientIDB, connectionIDB, commitmenttypes.NewMerklePrefix([]byte(ibcexported.StoreKey))),
			[]*connectiontypes.Version{ibctesting.ConnectionVersion},
			0,
		)
		connEndBz, err := cfg.Codec.Marshal(&connEnd)
		s.Require().NoError(err)
		connHash := sha256.Sum256(connEndBz)

		proofAck := s.createPacketAttestationProof(proofHeight, connPath, connHash[:])

		msgConnOpenConfirm := connectiontypes.NewMsgConnectionOpenConfirm(
			connectionIDB,
			proofAck,
			clienttypes.NewHeight(0, proofHeight),
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgConnOpenConfirm)
		s.AssertTxSuccess(txResp)
	})

	channelVersion := transfertypes.V1

	t.Run("channel open init", func(t *testing.T) {
		msgChanOpenInit := channeltypes.NewMsgChannelOpenInit(
			transfertypes.PortID, channelVersion,
			channeltypes.UNORDERED, []string{connectionIDA},
			transfertypes.PortID, rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenInit)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenInitRes))
		channelIDA = msgChanOpenInitRes.ChannelId
	})

	t.Run("channel open try", func(t *testing.T) {
		chanPath := s.prefixedPath(host.ChannelKey(transfertypes.PortID, channelIDA))

		chanEnd := channeltypes.NewChannel(
			channeltypes.INIT,
			channeltypes.UNORDERED,
			channeltypes.NewCounterparty(transfertypes.PortID, ""),
			[]string{connectionIDA},
			channelVersion,
		)
		chanEndBz, err := cfg.Codec.Marshal(&chanEnd)
		s.Require().NoError(err)
		chanHash := sha256.Sum256(chanEndBz)

		proofInit := s.createPacketAttestationProof(proofHeight, chanPath, chanHash[:])

		msgChanOpenTry := channeltypes.NewMsgChannelOpenTry(
			transfertypes.PortID, channelVersion,
			channeltypes.UNORDERED, []string{connectionIDB},
			transfertypes.PortID, channelIDA,
			channelVersion, proofInit, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenTry)
		s.AssertTxSuccess(txResp)

		s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &msgChanOpenTryRes))
		channelIDB = msgChanOpenTryRes.ChannelId
	})

	t.Run("channel open ack", func(t *testing.T) {
		chanPath := s.prefixedPath(host.ChannelKey(transfertypes.PortID, channelIDB))

		chanEnd := channeltypes.NewChannel(
			channeltypes.TRYOPEN,
			channeltypes.UNORDERED,
			channeltypes.NewCounterparty(transfertypes.PortID, channelIDA),
			[]string{connectionIDB},
			channelVersion,
		)
		chanEndBz, err := cfg.Codec.Marshal(&chanEnd)
		s.Require().NoError(err)
		chanHash := sha256.Sum256(chanEndBz)

		proofTry := s.createPacketAttestationProof(proofHeight, chanPath, chanHash[:])

		msgChanOpenAck := channeltypes.NewMsgChannelOpenAck(
			transfertypes.PortID, channelIDA,
			channelIDB, channelVersion,
			proofTry, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenAck)
		s.AssertTxSuccess(txResp)
	})

	t.Run("channel open confirm", func(t *testing.T) {
		chanPath := s.prefixedPath(host.ChannelKey(transfertypes.PortID, channelIDA))

		chanEnd := channeltypes.NewChannel(
			channeltypes.OPEN,
			channeltypes.UNORDERED,
			channeltypes.NewCounterparty(transfertypes.PortID, channelIDB),
			[]string{connectionIDA},
			channelVersion,
		)
		chanEndBz, err := cfg.Codec.Marshal(&chanEnd)
		s.Require().NoError(err)
		chanHash := sha256.Sum256(chanEndBz)

		proofAck := s.createPacketAttestationProof(proofHeight, chanPath, chanHash[:])

		msgChanOpenConfirm := channeltypes.NewMsgChannelOpenConfirm(
			transfertypes.PortID, channelIDB,
			proofAck, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgChanOpenConfirm)
		s.AssertTxSuccess(txResp)
	})

	t.Run("send IBC transfer with short timeout", func(t *testing.T) {
		// Use a timeout 5 seconds in the future
		timeoutTimestamp := uint64(time.Now().Add(5 * time.Second).UnixNano())
		txResp := s.Transfer(ctx, chainA, userAWallet, transfertypes.PortID, channelIDA, testvalues.DefaultTransferAmount(chainADenom), userAWallet.FormattedAddress(), userBWallet.FormattedAddress(), clienttypes.ZeroHeight(), timeoutTimestamp, "")
		s.AssertTxSuccess(txResp)

		var err error
		packet, err = ibctesting.ParseV1PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
		t.Logf("Packet sent with timeout: seq=%d, timeout=%d", packet.Sequence, packet.TimeoutTimestamp)

		// Wait for timeout to pass
		time.Sleep(6 * time.Second)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)
	})

	t.Run("timeout packet with attestation proof (non-membership)", func(t *testing.T) {
		// Update client with timestamp after packet timeout
		newTimestamp := uint64(time.Now().UnixNano())
		proofHeight++
		stateAttestation := &attestations.StateAttestation{
			Height:    proofHeight,
			Timestamp: newTimestamp,
		}
		stateAttestationData, err := proto.Marshal(stateAttestation)
		s.Require().NoError(err)

		signatures := s.signAttestationData(stateAttestationData)
		updateProof := &attestations.AttestationProof{
			AttestationData: stateAttestationData,
			Signatures:      signatures,
		}

		msgUpdateClient, err := clienttypes.NewMsgUpdateClient(clientIDA, updateProof, rlyWallet.FormattedAddress())
		s.Require().NoError(err)
		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgUpdateClient)
		s.AssertTxSuccess(txResp)

		receiptPath := s.prefixedPath(host.PacketReceiptKey(packet.DestinationPort, packet.DestinationChannel, packet.Sequence))

		proofUnreceived := s.createNonMembershipProof(proofHeight, receiptPath)

		msgTimeout := channeltypes.NewMsgTimeout(
			packet,
			1,
			proofUnreceived,
			clienttypes.NewHeight(0, proofHeight),
			rlyWallet.FormattedAddress(),
		)

		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgTimeout)
		s.AssertTxSuccess(txResp)
		t.Log("Packet timed out")
	})

	t.Run("verify tokens refunded after timeout", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainA, transfertypes.PortID, channelIDA, 1)

		actualBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
		t.Logf("User A refunded after timeout: %d", actualBalance)
	})
}
