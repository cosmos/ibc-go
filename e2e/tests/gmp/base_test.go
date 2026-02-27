//go:build !test_e2e

package gmp

import (
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"
	"github.com/ethereum/go-ethereum/crypto"
	testifysuite "github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	gmptypes "github.com/cosmos/ibc-go/v10/modules/apps/27-gmp/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	hostv2 "github.com/cosmos/ibc-go/v10/modules/core/24-host/v2"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	"github.com/cosmos/ibc-go/v10/modules/light-clients/attestations"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

const (
	testSalt        = "test-salt"
	numAttestors    = 3
	quorumThreshold = 2
	proofHeight     = 100
)

func TestGMPTestSuite(t *testing.T) {
	testifysuite.Run(t, new(GMPTestSuite))
}

type GMPTestSuite struct {
	testsuite.E2ETestSuite
	attestorKeys []*ecdsa.PrivateKey
}

func (s *GMPTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 1, nil)
	s.setupAttestors()
}

// TestMsgSendCall_BankTransfer tests the full GMP flow using attestations light clients:
// 1. Create two attestations clients on a single chain
// 2. Send MsgSendCall to create GMP packet
// 3. Relay packet with attestation proof
// 4. Verify bank transfer executed on destination
func (s *GMPTestSuite) TestMsgSendCall_BankTransfer() {
	t := s.T()
	ctx := context.TODO()

	chain := s.GetAllChains()[0]
	chainDenom := chain.Config().Denom

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	senderWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	recipientWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		clientIDA, clientIDB string
		packet               channeltypesv2.Packet
		ack                  channeltypesv2.Acknowledgement
		gmpAccountAddr       string
		initialBalance       sdkmath.Int
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chain), "failed to wait for blocks")

	proofTimestamp := uint64(time.Now().UnixNano())

	t.Run("create attestations clients", func(t *testing.T) {
		clientIDA = s.createAttestationsClient(ctx, chain, rlyWallet, proofTimestamp)
		clientIDB = s.createAttestationsClient(ctx, chain, rlyWallet, proofTimestamp)
		t.Logf("Created clients: %s, %s", clientIDA, clientIDB)
	})

	t.Run("verify client status", func(t *testing.T) {
		for _, clientID := range []string{clientIDA, clientIDB} {
			status, err := query.ClientStatus(ctx, chain, clientID)
			s.Require().NoError(err)
			s.Require().Equal(ibcexported.Active.String(), status)
		}
	})

	t.Run("register counterparties", func(t *testing.T) {
		s.registerCounterparty(ctx, chain, rlyWallet, clientIDA, clientIDB)
		s.registerCounterparty(ctx, chain, rlyWallet, clientIDB, clientIDA)
		t.Logf("Registered counterparties: %s <-> %s", clientIDA, clientIDB)
	})

	t.Run("get initial balance", func(t *testing.T) {
		balance, err := query.Balance(ctx, chain, recipientWallet.FormattedAddress(), chainDenom)
		s.Require().NoError(err)
		initialBalance = balance
	})

	t.Run("compute and fund GMP account", func(t *testing.T) {
		// GMP account is derived from destination client, sender, and salt
		accountID := gmptypes.NewAccountIdentifier(clientIDB, senderWallet.FormattedAddress(), []byte(testSalt))
		addr, err := gmptypes.BuildAddressPredictable(&accountID)
		s.Require().NoError(err)
		gmpAccountAddr = addr.String()

		msgSend := &banktypes.MsgSend{
			FromAddress: rlyWallet.FormattedAddress(),
			ToAddress:   gmpAccountAddr,
			Amount:      sdk.NewCoins(sdk.NewCoin(chainDenom, sdkmath.NewInt(testvalues.StartingTokenAmount))),
		}
		txResp := s.BroadcastMessages(ctx, chain, rlyWallet, msgSend)
		s.AssertTxSuccess(txResp)
		t.Logf("GMP account: %s", gmpAccountAddr)
	})

	t.Run("send MsgSendCall", func(t *testing.T) {
		msgSend := &banktypes.MsgSend{
			FromAddress: gmpAccountAddr,
			ToAddress:   recipientWallet.FormattedAddress(),
			Amount:      sdk.NewCoins(sdk.NewCoin(chainDenom, sdkmath.NewInt(testvalues.IBCTransferAmount))),
		}

		payload, err := gmptypes.SerializeCosmosTx(testsuite.Codec(), []proto.Message{msgSend})
		s.Require().NoError(err)

		msgSendCall := gmptypes.NewMsgSendCall(
			clientIDA,
			senderWallet.FormattedAddress(),
			"",
			payload,
			[]byte(testSalt),
			uint64(time.Now().Add(10*time.Minute).Unix()),
			gmptypes.EncodingProtobuf,
			"",
		)

		txResp := s.BroadcastMessages(ctx, chain, senderWallet, msgSendCall)
		s.AssertTxSuccess(txResp)

		packet, err = ibctesting.ParseV2PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		t.Logf("Packet sent: seq=%d", packet.Sequence)
	})

	t.Run("recv packet", func(t *testing.T) {
		commitment := channeltypesv2.CommitPacket(packet)
		path := s.hashPath(hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence))
		proof := s.createAttestationProof(path, commitment)

		msgRecvPacket := channeltypesv2.NewMsgRecvPacket(
			packet, proof, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chain, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)

		ackBz, err := ibctesting.ParseAckV2FromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NoError(proto.Unmarshal(ackBz, &ack))
	})

	t.Run("acknowledge packet", func(t *testing.T) {
		commitment := channeltypesv2.CommitAcknowledgement(ack)
		path := s.hashPath(hostv2.PacketAcknowledgementKey(packet.DestinationClient, packet.Sequence))
		proof := s.createAttestationProof(path, commitment)

		msgAck := channeltypesv2.NewMsgAcknowledgement(
			packet, ack, proof, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chain, rlyWallet, msgAck)
		s.AssertTxSuccess(txResp)
	})

	t.Run("verify transfer", func(t *testing.T) {
		balance, err := query.Balance(ctx, chain, recipientWallet.FormattedAddress(), chainDenom)
		s.Require().NoError(err)

		expected := initialBalance.Add(sdkmath.NewInt(testvalues.IBCTransferAmount))
		s.Require().Equal(expected, balance)
		t.Logf("Recipient balance: %s -> %s", initialBalance, balance)
	})
}

func (s *GMPTestSuite) setupAttestors() {
	for range numAttestors {
		key, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.attestorKeys = append(s.attestorKeys, key)
	}
}

func (s *GMPTestSuite) getAttestorAddresses() []string {
	addresses := make([]string, len(s.attestorKeys))
	for i, key := range s.attestorKeys {
		addresses[i] = crypto.PubkeyToAddress(key.PublicKey).Hex()
	}
	return addresses
}

func (s *GMPTestSuite) createAttestationsClient(ctx context.Context, chain ibc.Chain, wallet ibc.Wallet, timestamp uint64) string {
	clientState := attestations.NewClientState(s.getAttestorAddresses(), quorumThreshold, proofHeight)
	consensusState := &attestations.ConsensusState{Timestamp: timestamp}

	msg, err := clienttypes.NewMsgCreateClient(clientState, consensusState, wallet.FormattedAddress())
	s.Require().NoError(err)

	txResp := s.BroadcastMessages(ctx, chain, wallet, msg)
	s.AssertTxSuccess(txResp)

	var res clienttypes.MsgCreateClientResponse
	s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &res))
	return res.ClientId
}

func (s *GMPTestSuite) registerCounterparty(ctx context.Context, chain ibc.Chain, wallet ibc.Wallet, clientID, counterpartyID string) {
	msg := clientv2types.NewMsgRegisterCounterparty(
		clientID,
		[][]byte{[]byte("")},
		counterpartyID,
		wallet.FormattedAddress(),
	)
	txResp := s.BroadcastMessages(ctx, chain, wallet, msg)
	s.AssertTxSuccess(txResp)
}

func (*GMPTestSuite) hashPath(key []byte) []byte {
	return crypto.Keccak256(key)
}

func (s *GMPTestSuite) createAttestationProof(path, commitment []byte) []byte {
	attestation := &attestations.PacketAttestation{
		Height:  proofHeight,
		Packets: []attestations.PacketCompact{{Path: path, Commitment: commitment}},
	}
	data, err := attestation.ABIEncode()
	s.Require().NoError(err)

	hash := attestations.TaggedSigningInput(data, attestations.AttestationTypePacket)
	signatures := make([][]byte, len(s.attestorKeys))
	for i, key := range s.attestorKeys {
		sig, err := crypto.Sign(hash[:], key)
		s.Require().NoError(err)
		signatures[i] = sig
	}

	proof := &attestations.AttestationProof{AttestationData: data, Signatures: signatures}
	proofBz, err := proto.Marshal(proof)
	s.Require().NoError(err)
	return proofBz
}

func (s *GMPTestSuite) TestQueryAccountAddress() {
	ctx := context.TODO()
	chain := s.GetAllChains()[0]

	senderWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chain), "failed to wait for blocks")

	req := &gmptypes.QueryAccountAddressRequest{
		ClientId: ibctesting.FirstClientID,
		Sender:   senderWallet.FormattedAddress(),
		Salt:     "",
	}

	resp, err := query.GRPCQuery[gmptypes.QueryAccountAddressResponse](ctx, chain, req)
	s.Require().NoError(err)
	s.Require().NotEmpty(resp.AccountAddress)

	accountID := gmptypes.NewAccountIdentifier(ibctesting.FirstClientID, senderWallet.FormattedAddress(), []byte{})
	expectedAddr, err := gmptypes.BuildAddressPredictable(&accountID)
	s.Require().NoError(err)
	s.Require().Equal(expectedAddr.String(), resp.AccountAddress)
}

func (s *GMPTestSuite) TestQueryAccountIdentifier() {
	t := s.T()
	ctx := context.TODO()
	chain := s.GetAllChains()[0]

	rlyWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	senderWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	recipientWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)

	var (
		clientIDA, clientIDB string
		gmpAccountAddr       string
	)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chain), "failed to wait for blocks")

	proofTimestamp := uint64(time.Now().UnixNano())

	t.Run("setup clients and counterparties", func(t *testing.T) {
		clientIDA = s.createAttestationsClient(ctx, chain, rlyWallet, proofTimestamp)
		clientIDB = s.createAttestationsClient(ctx, chain, rlyWallet, proofTimestamp)
		s.registerCounterparty(ctx, chain, rlyWallet, clientIDA, clientIDB)
		s.registerCounterparty(ctx, chain, rlyWallet, clientIDB, clientIDA)
	})

	t.Run("create GMP account via packet", func(t *testing.T) {
		accountID := gmptypes.NewAccountIdentifier(clientIDB, senderWallet.FormattedAddress(), []byte(testSalt))
		addr, err := gmptypes.BuildAddressPredictable(&accountID)
		s.Require().NoError(err)
		gmpAccountAddr = addr.String()

		msgSend := &banktypes.MsgSend{
			FromAddress: rlyWallet.FormattedAddress(),
			ToAddress:   gmpAccountAddr,
			Amount:      sdk.NewCoins(sdk.NewCoin(chain.Config().Denom, sdkmath.NewInt(testvalues.StartingTokenAmount))),
		}
		txResp := s.BroadcastMessages(ctx, chain, rlyWallet, msgSend)
		s.AssertTxSuccess(txResp)

		payload, err := gmptypes.SerializeCosmosTx(testsuite.Codec(), []proto.Message{
			&banktypes.MsgSend{
				FromAddress: gmpAccountAddr,
				ToAddress:   recipientWallet.FormattedAddress(),
				Amount:      sdk.NewCoins(sdk.NewCoin(chain.Config().Denom, sdkmath.NewInt(testvalues.IBCTransferAmount))),
			},
		})
		s.Require().NoError(err)

		msgSendCall := gmptypes.NewMsgSendCall(
			clientIDA,
			senderWallet.FormattedAddress(),
			"",
			payload,
			[]byte(testSalt),
			uint64(time.Now().Add(10*time.Minute).Unix()),
			gmptypes.EncodingProtobuf,
			"",
		)

		txResp = s.BroadcastMessages(ctx, chain, senderWallet, msgSendCall)
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV2PacketFromEvents(txResp.Events)
		s.Require().NoError(err)

		commitment := channeltypesv2.CommitPacket(packet)
		path := s.hashPath(hostv2.PacketCommitmentKey(packet.SourceClient, packet.Sequence))
		proof := s.createAttestationProof(path, commitment)

		msgRecvPacket := channeltypesv2.NewMsgRecvPacket(
			packet, proof, clienttypes.NewHeight(0, proofHeight), rlyWallet.FormattedAddress(),
		)

		txResp = s.BroadcastMessages(ctx, chain, rlyWallet, msgRecvPacket)
		s.AssertTxSuccess(txResp)
	})

	t.Run("query account identifier", func(t *testing.T) {
		req := &gmptypes.QueryAccountIdentifierRequest{
			AccountAddress: gmpAccountAddr,
		}

		resp, err := query.GRPCQuery[gmptypes.QueryAccountIdentifierResponse](ctx, chain, req)
		s.Require().NoError(err)
		s.Require().Equal(clientIDB, resp.AccountId.ClientId)
		s.Require().Equal(senderWallet.FormattedAddress(), resp.AccountId.Sender)
		s.Require().Equal([]byte(testSalt), resp.AccountId.Salt)
	})
}
