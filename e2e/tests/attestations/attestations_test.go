//go:build !test_e2e

package attestations

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"strconv"
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
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
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

func (s *AttestationsTestSuite) TestMsgTransfer_Attestations() {
	t := s.T()
	ctx := context.TODO()

	chains := s.GetAllChains()
	chainA := chains[0]
	chainADenom := chainA.Config().Denom

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		clientIDA      string
		clientIDB      string
		packet         channeltypesv2.Packet
		ack            channeltypesv2.Acknowledgement
		proofHeight    uint64 = 100
		proofTimestamp uint64
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

	t.Run("register counterparties for IBC v2", func(t *testing.T) {
		msgRegisterCounterpartyA := clientv2types.NewMsgRegisterCounterparty(
			clientIDA,
			[][]byte{[]byte("")},
			clientIDB,
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgRegisterCounterpartyA)
		s.AssertTxSuccess(txResp)

		msgRegisterCounterpartyB := clientv2types.NewMsgRegisterCounterparty(
			clientIDB,
			[][]byte{[]byte("")},
			clientIDA,
			rlyWallet.FormattedAddress(),
		)

		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgRegisterCounterpartyB)
		s.AssertTxSuccess(txResp)

		t.Logf("Registered counterparties: %s <-> %s", clientIDA, clientIDB)
	})

	t.Run("send IBC v2 transfer", func(t *testing.T) {
		token := transfertypes.Token{
			Denom:  transfertypes.NewDenom(chainADenom),
			Amount: strconv.FormatInt(testvalues.IBCTransferAmount, 10),
		}
		packetData := transfertypes.NewFungibleTokenPacketData(
			token.Denom.Path(),
			token.Amount,
			userAWallet.FormattedAddress(),
			userBWallet.FormattedAddress(),
			"",
		)

		data, err := json.Marshal(packetData)
		s.Require().NoError(err)

		payload := channeltypesv2.NewPayload(
			transfertypes.PortID, transfertypes.PortID,
			transfertypes.V1, transfertypes.EncodingJSON, data,
		)

		timeoutTimestamp := uint64(time.Now().Add(10 * time.Minute).Unix())
		msgSendPacket := channeltypesv2.NewMsgSendPacket(
			clientIDA, timeoutTimestamp,
			userAWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSendPacket)
		s.AssertTxSuccess(txResp)

		packet, err = ibctesting.ParseV2PacketFromEvents(txResp.Events)
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
		packetCommitment := channeltypesv2.CommitPacket(packet)
		packetPath := s.hashPath(hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence))

		proofCommitment := s.createPacketAttestationProof(proofHeight, packetPath, packetCommitment)

		msgRecvPacket := channeltypesv2.NewMsgRecvPacket(packet, proofCommitment, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		ackBz, err := ibctesting.ParseAckV2FromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ackBz)

		err = proto.Unmarshal(ackBz, &ack)
		s.Require().NoError(err)
		t.Log("Packet received")
	})

	t.Run("acknowledge packet with attestation proof", func(t *testing.T) {
		ackCommitment := channeltypesv2.CommitAcknowledgement(ack)
		ackPath := s.hashPath(hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence))

		proofAcked := s.createPacketAttestationProof(proofHeight, ackPath, ackCommitment)

		msgAcknowledgement := channeltypesv2.NewMsgAcknowledgement(packet, ack, proofAcked, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
		s.AssertTxSuccess(txResp)
		t.Log("Packet acknowledged")
	})

	t.Run("verify tokens transferred", func(t *testing.T) {
		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, clientIDB)
		actualBalance, err := query.Balance(ctx, chainA, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
		t.Logf("User B received %d of %s", actualBalance.Int64(), ibcToken.IBCDenom())
	})

	t.Run("send IBC v2 transfer back (unwind)", func(t *testing.T) {
		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, clientIDB)

		token := transfertypes.Token{
			Denom:  transfertypes.ExtractDenomFromPath(ibcToken.Path()),
			Amount: strconv.FormatInt(testvalues.IBCTransferAmount, 10),
		}
		packetData := transfertypes.NewFungibleTokenPacketData(
			token.Denom.Path(),
			token.Amount,
			userBWallet.FormattedAddress(),
			userAWallet.FormattedAddress(),
			"",
		)

		data, err := json.Marshal(packetData)
		s.Require().NoError(err)

		payload := channeltypesv2.NewPayload(
			transfertypes.PortID, transfertypes.PortID,
			transfertypes.V1, transfertypes.EncodingJSON, data,
		)

		timeoutTimestamp := uint64(time.Now().Add(10 * time.Minute).Unix())
		msgSendPacket := channeltypesv2.NewMsgSendPacket(
			clientIDB, timeoutTimestamp,
			userBWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainA, userBWallet, msgSendPacket)
		s.AssertTxSuccess(txResp)

		packet, err = ibctesting.ParseV2PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
		t.Logf("Return packet sent: seq=%d", packet.Sequence)
	})

	t.Run("recv return packet", func(t *testing.T) {
		packetCommitment := channeltypesv2.CommitPacket(packet)
		packetPath := s.hashPath(hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence))

		proofCommitment := s.createPacketAttestationProof(proofHeight, packetPath, packetCommitment)

		msgRecvPacket := channeltypesv2.NewMsgRecvPacket(packet, proofCommitment, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		ackBz, err := ibctesting.ParseAckV2FromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ackBz)

		err = proto.Unmarshal(ackBz, &ack)
		s.Require().NoError(err)
		t.Log("Return packet received")
	})

	t.Run("acknowledge return packet", func(t *testing.T) {
		ackCommitment := channeltypesv2.CommitAcknowledgement(ack)
		ackPath := s.hashPath(hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence))

		proofAcked := s.createPacketAttestationProof(proofHeight, ackPath, ackCommitment)

		msgAcknowledgement := channeltypesv2.NewMsgAcknowledgement(packet, ack, proofAcked, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress())

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgAcknowledgement)
		s.AssertTxSuccess(txResp)
		t.Log("Return packet acknowledged")
	})

	t.Run("verify tokens unwound", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
		t.Logf("User A recovered full balance: %d", actualBalance)

		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, clientIDB)
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

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		clientIDA      string
		clientIDB      string
		packet         channeltypesv2.Packet
		proofHeight    uint64 = 100
		proofTimestamp uint64
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

	t.Run("register counterparties for IBC v2", func(t *testing.T) {
		msgRegisterCounterpartyA := clientv2types.NewMsgRegisterCounterparty(
			clientIDA,
			[][]byte{[]byte("")},
			clientIDB,
			rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWallet, msgRegisterCounterpartyA)
		s.AssertTxSuccess(txResp)

		msgRegisterCounterpartyB := clientv2types.NewMsgRegisterCounterparty(
			clientIDB,
			[][]byte{[]byte("")},
			clientIDA,
			rlyWallet.FormattedAddress(),
		)

		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgRegisterCounterpartyB)
		s.AssertTxSuccess(txResp)
	})

	t.Run("send IBC v2 transfer with short timeout", func(t *testing.T) {
		token := transfertypes.Token{
			Denom:  transfertypes.NewDenom(chainADenom),
			Amount: strconv.FormatInt(testvalues.IBCTransferAmount, 10),
		}
		packetData := transfertypes.NewFungibleTokenPacketData(
			token.Denom.Path(),
			token.Amount,
			userAWallet.FormattedAddress(),
			userBWallet.FormattedAddress(),
			"",
		)

		data, err := json.Marshal(packetData)
		s.Require().NoError(err)

		payload := channeltypesv2.NewPayload(
			transfertypes.PortID, transfertypes.PortID,
			transfertypes.V1, transfertypes.EncodingJSON, data,
		)

		// Timeout 5 seconds in the future (IBC v2 uses seconds, not nanoseconds)
		timeoutTimestamp := uint64(time.Now().Add(5 * time.Second).Unix())
		msgSendPacket := channeltypesv2.NewMsgSendPacket(
			clientIDA, timeoutTimestamp,
			userAWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSendPacket)
		s.AssertTxSuccess(txResp)

		packet, err = ibctesting.ParseV2PacketFromEvents(txResp.Events)
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
		stateAttestationData, err := stateAttestation.ABIEncode()
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

		receiptPath := s.hashPath(hostv2.PacketReceiptKey(packet.DestinationClient, packet.Sequence))

		proofUnreceived := s.createNonMembershipProof(proofHeight, receiptPath)

		msgTimeout := channeltypesv2.NewMsgTimeout(
			packet,
			proofUnreceived,
			clienttypes.NewHeight(0, proofHeight),
			rlyWallet.FormattedAddress(),
		)

		txResp = s.BroadcastMessages(ctx, chainA, rlyWallet, msgTimeout)
		s.AssertTxSuccess(txResp)
		t.Log("Packet timed out")
	})

	t.Run("verify tokens refunded after timeout", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance)
		t.Logf("User A refunded after timeout: %d", actualBalance)
	})
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

// hashPath hashes the key path using keccak256.
// The attestations client uses counterparty merkle prefix [][]byte{[]byte("")},
// meaning paths contain only the ICS24 key without any store prefix.
func (*AttestationsTestSuite) hashPath(key []byte) []byte {
	return crypto.Keccak256(key)
}

func (s *AttestationsTestSuite) createPacketAttestationProof(height uint64, path []byte, commitment []byte) []byte {
	hashedCommitment := crypto.Keccak256(commitment)
	packetAttestation := &attestations.PacketAttestation{
		Height: height,
		Packets: []attestations.PacketCompact{
			{
				Path:       path,
				Commitment: hashedCommitment,
			},
		},
	}
	attestationData, err := packetAttestation.ABIEncode()
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
	attestationData, err := packetAttestation.ABIEncode()
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
