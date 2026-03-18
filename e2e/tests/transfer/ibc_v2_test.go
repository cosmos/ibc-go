//go:build !test_e2e

package transfer

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/interchaintest/v10/ibc"
	test "github.com/cosmos/interchaintest/v10/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types/v2"
	ibctmtypes "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func TestTransferTestSuiteIBCV2(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuiteIBCV2))
}

type TransferTestSuiteIBCV2 struct {
	testsuite.E2ETestSuite
}

func (s *TransferTestSuiteIBCV2) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

func (s *TransferTestSuiteIBCV2) TestMsgTransfer_Tendermint_IBCv2_ManualRelay() {
	t := s.T()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	chainADenom := chainA.Config().Denom

	rlyWalletA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	rlyWalletB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userAWallet := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userBWallet := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks")

	var (
		clientIDA string
		clientIDB string
		packet    channeltypesv2.Packet
		ack       channeltypesv2.Acknowledgement
	)

	t.Run("create tendermint clients", func(t *testing.T) {
		clientIDA = s.createTendermintClient(ctx, chainA, chainB, rlyWalletA)
		clientIDB = s.createTendermintClient(ctx, chainB, chainA, rlyWalletB)
	})

	t.Run("register counterparties", func(t *testing.T) {
		prefix := commitmenttypesv2.NewMerklePath([]byte("ibc"), []byte(""))

		msgRegisterA := clientv2types.NewMsgRegisterCounterparty(
			clientIDA,
			prefix.KeyPath,
			clientIDB,
			rlyWalletA.FormattedAddress(),
		)
		txResp := s.BroadcastMessages(ctx, chainA, rlyWalletA, msgRegisterA)
		s.AssertTxSuccess(txResp)

		msgRegisterB := clientv2types.NewMsgRegisterCounterparty(
			clientIDB,
			prefix.KeyPath,
			clientIDA,
			rlyWalletB.FormattedAddress(),
		)
		txResp = s.BroadcastMessages(ctx, chainB, rlyWalletB, msgRegisterB)
		s.AssertTxSuccess(txResp)
	})

	t.Run("send ibc v2 transfer", func(t *testing.T) {
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
		msgSend := channeltypesv2.NewMsgSendPacket(
			clientIDA, timeoutTimestamp,
			userAWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSend)
		s.AssertTxSuccess(txResp)

		packet, err = ibctesting.ParseV2PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
	})

	t.Run("tokens escrowed on chainA", func(t *testing.T) {
		actual, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actual)
	})

	t.Run("recv packet on chainB", func(t *testing.T) {
		proof, proofHeight := s.queryPacketCommitmentProof(ctx, chainA, packet.SourceClient, packet.Sequence)

		msgRecv := channeltypesv2.NewMsgRecvPacket(
			packet, proof, proofHeight, rlyWalletB.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainB, rlyWalletB, msgRecv)
		s.AssertTxSuccess(txResp)

		ackBz, err := ibctesting.ParseAckV2FromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ackBz)

		s.Require().NoError(proto.Unmarshal(ackBz, &ack))
	})

	t.Run("acknowledge on chainA", func(t *testing.T) {
		proof, proofHeight := s.queryPacketAcknowledgementProof(ctx, chainB, packet.DestinationClient, packet.Sequence)

		msgAck := channeltypesv2.NewMsgAcknowledgement(
			packet, ack, proof, proofHeight, rlyWalletA.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWalletA, msgAck)
		s.AssertTxSuccess(txResp)
	})

	t.Run("verify tokens received on chainB", func(t *testing.T) {
		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, clientIDB)
		balance, err := query.Balance(ctx, chainB, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)

		s.Require().Equal(testvalues.IBCTransferAmount, balance.Int64())
	})

	t.Run("return transfer from chainB", func(t *testing.T) {
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
		msgSend := channeltypesv2.NewMsgSendPacket(
			clientIDB, timeoutTimestamp,
			userBWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainB, userBWallet, msgSend)
		s.AssertTxSuccess(txResp)

		packet, err := ibctesting.ParseV2PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
	})

	t.Run("recv return packet on chainA", func(t *testing.T) {
		proof, proofHeight := s.queryPacketCommitmentProof(ctx, chainB, packet.SourceClient, packet.Sequence)

		msgRecv := channeltypesv2.NewMsgRecvPacket(
			packet, proof, proofHeight, rlyWalletA.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWalletA, msgRecv)
		s.AssertTxSuccess(txResp)

		ackBz, err := ibctesting.ParseAckV2FromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ackBz)

		s.Require().NoError(proto.Unmarshal(ackBz, &ack))
	})

	t.Run("acknowledge return on chainB", func(t *testing.T) {
		proof, proofHeight := s.queryPacketAcknowledgementProof(ctx, chainA, packet.DestinationClient, packet.Sequence)

		msgAck := channeltypesv2.NewMsgAcknowledgement(
			packet, ack, proof, proofHeight, rlyWalletB.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainB, rlyWalletB, msgAck)
		s.AssertTxSuccess(txResp)
	})

	t.Run("verify balances restored", func(t *testing.T) {
		balanceA, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, balanceA)

		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, clientIDB)
		balanceB, err := query.Balance(ctx, chainB, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(balanceB.Int64())
	})
}

func (s *TransferTestSuiteIBCV2) createTendermintClient(ctx context.Context, hostingChain, counterparty ibc.Chain, signer ibc.Wallet) string {
	latestBlock, err := query.GRPCQuery[cmtservice.GetLatestBlockResponse](ctx, counterparty, &cmtservice.GetLatestBlockRequest{})
	s.Require().NoError(err)
	s.Require().NotNil(latestBlock.Block)

	header := latestBlock.Block.Header

	stakingParams, err := query.GRPCQuery[stakingtypes.QueryParamsResponse](ctx, counterparty, &stakingtypes.QueryParamsRequest{})
	s.Require().NoError(err)

	unbondingTime := stakingParams.Params.UnbondingTime
	trustingPeriod := unbondingTime * 2 / 3
	maxClockDrift := 5 * time.Minute

	revisionNumber := clienttypes.ParseChainID(counterparty.Config().ChainID)

	latestHeight := clienttypes.NewHeight(revisionNumber, uint64(header.Height))
	clientState := ibctmtypes.NewClientState(
		counterparty.Config().ChainID,
		ibctmtypes.DefaultTrustLevel,
		trustingPeriod,
		unbondingTime,
		maxClockDrift,
		latestHeight,
		commitmenttypes.GetSDKSpecs(),
		ibctesting.UpgradePath,
	)

	consensusState := ibctmtypes.NewConsensusState(
		header.Time,
		commitmenttypes.NewMerkleRoot(header.AppHash),
		header.NextValidatorsHash,
	)

	msg, err := clienttypes.NewMsgCreateClient(clientState, consensusState, signer.FormattedAddress())
	s.Require().NoError(err)

	txResp := s.BroadcastMessages(ctx, hostingChain, signer, msg)
	s.AssertTxSuccess(txResp)

	var createRes clienttypes.MsgCreateClientResponse
	s.Require().NoError(testsuite.UnmarshalMsgResponses(txResp, &createRes))
	return createRes.ClientId
}

func (s *TransferTestSuiteIBCV2) queryPacketCommitmentProof(ctx context.Context, chain ibc.Chain, clientID string, sequence uint64) ([]byte, clienttypes.Height) {
	res, err := query.GRPCQuery[channeltypesv2.QueryPacketCommitmentResponse](ctx, chain, &channeltypesv2.QueryPacketCommitmentRequest{
		ClientId: clientID,
		Sequence: sequence,
	})
	s.Require().NoError(err)
	return res.Proof, res.ProofHeight
}

func (s *TransferTestSuiteIBCV2) queryPacketAcknowledgementProof(ctx context.Context, chain ibc.Chain, clientID string, sequence uint64) ([]byte, clienttypes.Height) {
	res, err := query.GRPCQuery[channeltypesv2.QueryPacketAcknowledgementResponse](ctx, chain, &channeltypesv2.QueryPacketAcknowledgementRequest{
		ClientId: clientID,
		Sequence: sequence,
	})
	s.Require().NoError(err)
	return res.Proof, res.ProofHeight
}
