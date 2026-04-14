//go:build !test_e2e

package transfer

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/interchaintest/v11/chain/cosmos"
	"github.com/cosmos/interchaintest/v11/ibc"
	test "github.com/cosmos/interchaintest/v11/testutil"
	testifysuite "github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	rpcclient "github.com/cometbft/cometbft/rpc/client"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	clientv2types "github.com/cosmos/ibc-go/v11/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v11/modules/core/04-channel/v2/types"
	commitmenttypes "github.com/cosmos/ibc-go/v11/modules/core/23-commitment/types"
	commitmenttypesv2 "github.com/cosmos/ibc-go/v11/modules/core/23-commitment/types/v2"
	hostv2 "github.com/cosmos/ibc-go/v11/modules/core/24-host/v2"
	ibctmtypes "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
	ibctesting "github.com/cosmos/ibc-go/v11/testing"
)

// compatibility:from_version: v10.5.1
func TestTransferTestSuiteIBCV2(t *testing.T) {
	testifysuite.Run(t, new(TransferTestSuiteIBCV2))
}

type TransferTestSuiteIBCV2 struct {
	testsuite.E2ETestSuite
}

func (s *TransferTestSuiteIBCV2) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

func (s *TransferTestSuiteIBCV2) TestMsgTransfer_IBCv2_Succeeds() {
	t := s.T()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	chainACosmos, ok := chainA.(*cosmos.CosmosChain)
	s.Require().True(ok)
	chainBCosmos, ok := chainB.(*cosmos.CosmosChain)
	s.Require().True(ok)
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

		payload := channeltypesv2.NewPayload(
			transfertypes.PortID, transfertypes.PortID,
			transfertypes.V1, transfertypes.EncodingJSON, packetData.GetBytes(),
		)

		timeoutTimestamp := uint64(time.Now().Add(10 * time.Minute).Unix())
		msgSend := channeltypesv2.NewMsgSendPacket(
			clientIDA, timeoutTimestamp,
			userAWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSend)
		s.AssertTxSuccess(txResp)

		var err error
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
		targetHeight := s.manualRelayTargetHeight(ctx, chainACosmos, chainB, clientIDB)
		s.updateTendermintClient(ctx, chainB, chainA, clientIDB, rlyWalletB, targetHeight)

		proof, proofHeight := s.queryPacketCommitmentProof(ctx, chainA, packet.SourceClient, packet.Sequence, targetHeight)
		s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

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
		targetHeight := s.manualRelayTargetHeight(ctx, chainBCosmos, chainA, clientIDA)
		s.updateTendermintClient(ctx, chainA, chainB, clientIDA, rlyWalletA, targetHeight)

		proof, proofHeight := s.queryPacketAcknowledgementProof(ctx, chainB, packet.DestinationClient, packet.Sequence, targetHeight)
		s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

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

		payload := channeltypesv2.NewPayload(
			transfertypes.PortID, transfertypes.PortID,
			transfertypes.V1, transfertypes.EncodingJSON, packetData.GetBytes(),
		)

		timeoutTimestamp := uint64(time.Now().Add(10 * time.Minute).Unix())
		msgSend := channeltypesv2.NewMsgSendPacket(
			clientIDB, timeoutTimestamp,
			userBWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainB, userBWallet, msgSend)
		s.AssertTxSuccess(txResp)

		var err error
		packet, err = ibctesting.ParseV2PacketFromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(packet)
	})

	t.Run("recv return packet on chainA", func(t *testing.T) {
		targetHeight := s.manualRelayTargetHeight(ctx, chainBCosmos, chainA, clientIDA)
		s.updateTendermintClient(ctx, chainA, chainB, clientIDA, rlyWalletA, targetHeight)

		proof, proofHeight := s.queryPacketCommitmentProof(ctx, chainB, packet.SourceClient, packet.Sequence, targetHeight)
		s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

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
		targetHeight := s.manualRelayTargetHeight(ctx, chainACosmos, chainB, clientIDB)
		s.updateTendermintClient(ctx, chainB, chainA, clientIDB, rlyWalletB, targetHeight)

		proof, proofHeight := s.queryPacketAcknowledgementProof(ctx, chainA, packet.DestinationClient, packet.Sequence, targetHeight)
		s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

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

func (s *TransferTestSuiteIBCV2) TestMsgTransfer_IBCv2_Fails_InvalidAddress() {
	t := s.T()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	chainACosmos, ok := chainA.(*cosmos.CosmosChain)
	s.Require().True(ok)
	chainBCosmos, ok := chainB.(*cosmos.CosmosChain)
	s.Require().True(ok)
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

	t.Run("send ibc v2 transfer to invalid receiver", func(t *testing.T) {
		token := transfertypes.Token{
			Denom:  transfertypes.NewDenom(chainADenom),
			Amount: strconv.FormatInt(testvalues.IBCTransferAmount, 10),
		}
		packetData := transfertypes.NewFungibleTokenPacketData(
			token.Denom.Path(),
			token.Amount,
			userAWallet.FormattedAddress(),
			testvalues.InvalidAddress,
			"",
		)

		payload := channeltypesv2.NewPayload(
			transfertypes.PortID, transfertypes.PortID,
			transfertypes.V1, transfertypes.EncodingJSON, packetData.GetBytes(),
		)

		timeoutTimestamp := uint64(time.Now().Add(10 * time.Minute).Unix())
		msgSend := channeltypesv2.NewMsgSendPacket(
			clientIDA, timeoutTimestamp,
			userAWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSend)
		s.AssertTxSuccess(txResp)

		var err error
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

	t.Run("recv packet returns universal error ack", func(t *testing.T) {
		targetHeight := s.manualRelayTargetHeight(ctx, chainACosmos, chainB, clientIDB)
		s.updateTendermintClient(ctx, chainB, chainA, clientIDB, rlyWalletB, targetHeight)

		proof, proofHeight := s.queryPacketCommitmentProof(ctx, chainA, packet.SourceClient, packet.Sequence, targetHeight)
		s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

		msgRecv := channeltypesv2.NewMsgRecvPacket(
			packet, proof, proofHeight, rlyWalletB.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainB, rlyWalletB, msgRecv)
		s.AssertTxSuccess(txResp)

		ackBz, err := ibctesting.ParseAckV2FromEvents(txResp.Events)
		s.Require().NoError(err)
		s.Require().NotNil(ackBz)

		s.Require().NoError(proto.Unmarshal(ackBz, &ack))
		s.Require().False(ack.Success())
		s.Require().Len(ack.AppAcknowledgements, 1)
		s.Require().Equal(channeltypesv2.ErrorAcknowledgement[:], ack.AppAcknowledgements[0])
	})

	t.Run("acknowledge on chainA and refund sender", func(t *testing.T) {
		targetHeight := s.manualRelayTargetHeight(ctx, chainBCosmos, chainA, clientIDA)
		s.updateTendermintClient(ctx, chainA, chainB, clientIDA, rlyWalletA, targetHeight)

		proof, proofHeight := s.queryPacketAcknowledgementProof(ctx, chainB, packet.DestinationClient, packet.Sequence, targetHeight)
		s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

		msgAck := channeltypesv2.NewMsgAcknowledgement(
			packet, ack, proof, proofHeight, rlyWalletA.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWalletA, msgAck)
		s.AssertTxSuccess(txResp)
	})

	t.Run("sender refunded and receiver gets no tokens", func(t *testing.T) {
		senderBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, senderBalance)

		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, clientIDB)
		receiverBalance, err := query.Balance(ctx, chainB, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(receiverBalance.Int64())
	})
}

func (s *TransferTestSuiteIBCV2) TestMsgTransfer_IBCv2_Timeout() {
	t := s.T()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	_, ok := chainA.(*cosmos.CosmosChain)
	s.Require().True(ok)
	chainBCosmos, ok := chainB.(*cosmos.CosmosChain)
	s.Require().True(ok)
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

	t.Run("send ibc v2 transfer with timeout", func(t *testing.T) {
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

		payload := channeltypesv2.NewPayload(
			transfertypes.PortID, transfertypes.PortID,
			transfertypes.V1, transfertypes.EncodingJSON, packetData.GetBytes(),
		)

		timeoutTimestamp := uint64(time.Now().Add(10 * time.Second).Unix())
		msgSend := channeltypesv2.NewMsgSendPacket(
			clientIDA, timeoutTimestamp,
			userAWallet.FormattedAddress(), payload,
		)

		txResp := s.BroadcastMessages(ctx, chainA, userAWallet, msgSend)
		s.AssertTxSuccess(txResp)

		var err error
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

	t.Run("wait for timeout to elapse", func(t *testing.T) {
		time.Sleep(11 * time.Second)
		s.Require().NoError(test.WaitForBlocks(ctx, 1, chainA, chainB), "failed to wait for blocks after timeout")
	})

	t.Run("timeout packet on source chain", func(t *testing.T) {
		targetHeight := s.manualRelayTargetHeight(ctx, chainBCosmos, chainA, clientIDA)
		s.updateTendermintClient(ctx, chainA, chainB, clientIDA, rlyWalletA, targetHeight)

		proofUnreceived, proofHeight := s.queryPacketReceiptProof(ctx, chainB, packet.DestinationClient, packet.Sequence, targetHeight)
		s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

		msgTimeout := channeltypesv2.NewMsgTimeout(
			packet,
			proofUnreceived,
			proofHeight,
			rlyWalletA.FormattedAddress(),
		)

		txResp := s.BroadcastMessages(ctx, chainA, rlyWalletA, msgTimeout)
		s.AssertTxSuccess(txResp)
	})

	t.Run("verify sender refunded and receiver got no tokens", func(t *testing.T) {
		senderBalance, err := s.GetChainANativeBalance(ctx, userAWallet)
		s.Require().NoError(err)
		s.Require().Equal(testvalues.StartingTokenAmount, senderBalance)

		ibcToken := testsuite.GetIBCToken(chainADenom, transfertypes.PortID, clientIDB)
		receiverBalance, err := query.Balance(ctx, chainB, userBWallet.FormattedAddress(), ibcToken.IBCDenom())
		s.Require().NoError(err)
		s.Require().Zero(receiverBalance.Int64())
	})
}

func (s *TransferTestSuiteIBCV2) createTendermintClient(ctx context.Context, hostingChain, counterparty ibc.Chain, signer ibc.Wallet) string {
	latestBlock, err := query.GRPCQuery[cmtservice.GetLatestBlockResponse](ctx, counterparty, &cmtservice.GetLatestBlockRequest{})
	s.Require().NoError(err)
	s.Require().NotNil(latestBlock.SdkBlock)

	header := latestBlock.SdkBlock.Header

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

func (s *TransferTestSuiteIBCV2) queryPacketCommitmentProof(ctx context.Context, chain ibc.Chain, clientID string, sequence uint64, targetHeight int64) ([]byte, clienttypes.Height) {
	proofKey := hostv2.PacketCommitmentKey(clientID, sequence)
	proof, proofHeight, err := s.queryProofForIBCStore(ctx, chain, proofKey, targetHeight)
	s.Require().NoError(err)
	return proof, proofHeight
}

func (s *TransferTestSuiteIBCV2) queryPacketAcknowledgementProof(ctx context.Context, chain ibc.Chain, clientID string, sequence uint64, targetHeight int64) ([]byte, clienttypes.Height) {
	proofKey := hostv2.PacketAcknowledgementKey(clientID, sequence)
	proof, proofHeight, err := s.queryProofForIBCStore(ctx, chain, proofKey, targetHeight)
	s.Require().NoError(err)
	return proof, proofHeight
}

func (s *TransferTestSuiteIBCV2) queryPacketReceiptProof(ctx context.Context, chain ibc.Chain, clientID string, sequence uint64, targetHeight int64) ([]byte, clienttypes.Height) {
	proofKey := hostv2.PacketReceiptKey(clientID, sequence)
	proof, proofHeight, err := s.queryProofForIBCStore(ctx, chain, proofKey, targetHeight)
	s.Require().NoError(err)
	return proof, proofHeight
}

func (*TransferTestSuiteIBCV2) queryProofForIBCStore(ctx context.Context, chain ibc.Chain, key []byte, targetHeight int64) ([]byte, clienttypes.Height, error) {
	cosmosChain, ok := chain.(*cosmos.CosmosChain)
	if !ok {
		return nil, clienttypes.Height{}, fmt.Errorf("expected *cosmos.CosmosChain, got %T", chain)
	}
	if targetHeight <= 1 {
		return nil, clienttypes.Height{}, fmt.Errorf("targetHeight must be > 1, got %d", targetHeight)
	}

	res, err := cosmosChain.GetNode().Client.ABCIQueryWithOptions(ctx, "store/ibc/key", key, rpcclient.ABCIQueryOptions{Height: targetHeight - 1, Prove: true})
	if err != nil {
		return nil, clienttypes.Height{}, err
	}
	if res.Response.Code != 0 {
		return nil, clienttypes.Height{}, fmt.Errorf("abci query failed with code %d: %s", res.Response.Code, res.Response.Log)
	}

	merkleProof, err := commitmenttypes.ConvertProofs(res.Response.ProofOps)
	if err != nil {
		return nil, clienttypes.Height{}, err
	}

	proofBz, err := merkleProof.Marshal()
	if err != nil {
		return nil, clienttypes.Height{}, err
	}

	revision := clienttypes.ParseChainID(chain.Config().ChainID)
	proofHeight := clienttypes.NewHeight(revision, uint64(res.Response.Height)+1)

	return proofBz, proofHeight, nil
}

func (s *TransferTestSuiteIBCV2) updateTendermintClient(ctx context.Context, hostingChain, counterparty ibc.Chain, clientID string, signer ibc.Wallet, targetHeight int64) {
	hostedClientState, err := query.ClientState(ctx, hostingChain, clientID)
	s.Require().NoError(err)

	tmClientState, ok := hostedClientState.(*ibctmtypes.ClientState)
	s.Require().True(ok)

	trustedHeight := tmClientState.LatestHeight

	counterpartyChain, ok := counterparty.(*cosmos.CosmosChain)
	s.Require().True(ok)
	s.Require().Greater(uint64(targetHeight), trustedHeight.GetRevisionHeight())

	commitRes, err := counterpartyChain.GetNode().Client.Commit(ctx, &targetHeight)
	s.Require().NoError(err)
	s.Require().NotNil(commitRes.SignedHeader)

	validatorSet := s.queryValidatorSet(ctx, counterpartyChain, targetHeight)
	trustedValidators := s.queryValidatorSet(ctx, counterpartyChain, int64(trustedHeight.GetRevisionHeight()))

	validatorSetProto, err := validatorSet.ToProto()
	s.Require().NoError(err)
	validatorSetProto.TotalVotingPower = validatorSet.TotalVotingPower()

	trustedValidatorSetProto, err := trustedValidators.ToProto()
	s.Require().NoError(err)
	trustedValidatorSetProto.TotalVotingPower = trustedValidators.TotalVotingPower()

	tmHeader := &ibctmtypes.Header{
		SignedHeader:      commitRes.ToProto(),
		ValidatorSet:      validatorSetProto,
		TrustedHeight:     trustedHeight,
		TrustedValidators: trustedValidatorSetProto,
	}

	msgUpdateClient, err := clienttypes.NewMsgUpdateClient(clientID, tmHeader, signer.FormattedAddress())
	s.Require().NoError(err)

	txResp := s.BroadcastMessages(ctx, hostingChain, signer, msgUpdateClient)
	s.AssertTxSuccess(txResp)
}

func (s *TransferTestSuiteIBCV2) manualRelayTargetHeight(ctx context.Context, sourceChain *cosmos.CosmosChain, hostingChain ibc.Chain, clientID string) int64 {
	hostedClientState, err := query.ClientState(ctx, hostingChain, clientID)
	s.Require().NoError(err)

	tmClientState, ok := hostedClientState.(*ibctmtypes.ClientState)
	s.Require().True(ok)

	targetHeight, err := sourceChain.Height(ctx)
	s.Require().NoError(err)

	if uint64(targetHeight) <= tmClientState.LatestHeight.GetRevisionHeight() {
		s.Require().NoError(test.WaitForBlocks(ctx, 1, sourceChain))
		targetHeight, err = sourceChain.Height(ctx)
		s.Require().NoError(err)
	}

	s.Require().Greater(uint64(targetHeight), tmClientState.LatestHeight.GetRevisionHeight())
	return targetHeight
}

func (s *TransferTestSuiteIBCV2) queryValidatorSet(ctx context.Context, chain *cosmos.CosmosChain, height int64) *cmttypes.ValidatorSet {
	validators := make([]*cmttypes.Validator, 0)
	page := 1
	perPage := 100

	for {
		validatorsRes, err := chain.GetNode().Client.Validators(ctx, &height, &page, &perPage)
		s.Require().NoError(err)

		validators = append(validators, validatorsRes.Validators...)
		if len(validators) >= validatorsRes.Total {
			break
		}

		page++
	}

	return cmttypes.NewValidatorSet(validators)
}
