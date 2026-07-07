//go:build !test_e2e

package ratelimiting

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

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	rpcclient "github.com/cometbft/cometbft/rpc/client"
	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	ratelimitingtypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"
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

type RateLimV2TestSuite struct {
	testsuite.E2ETestSuite
}

// compatibility:from_version: v11.2.0
func TestRateLimitV2Suite(t *testing.T) {
	testifysuite.Run(t, new(RateLimV2TestSuite))
}

func (s *RateLimV2TestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

func (s *RateLimV2TestSuite) TestRateLimitIBCV2() {
	t := s.T()
	ctx := context.TODO()

	chainA, chainB := s.GetChains()
	chainACosmos, ok := chainA.(*cosmos.CosmosChain)
	s.Require().True(ok)
	chainBCosmos, ok := chainB.(*cosmos.CosmosChain)
	s.Require().True(ok)

	denomA := chainA.Config().Denom
	rlyWalletA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	rlyWalletB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

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
		s.registerCounterparties(ctx, chainA, chainB, rlyWalletA, rlyWalletB, clientIDA, clientIDB)
	})

	authorityA, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainA)
	s.Require().NoError(err)
	authorityB, err := query.ModuleAccountAddress(ctx, govtypes.ModuleName, chainB)
	s.Require().NoError(err)

	ibcTokenB := testsuite.GetIBCToken(denomA, transfertypes.PortID, clientIDB)

	t.Run("add outgoing v2 rate limit on chainA", func(t *testing.T) {
		s.addRateLimit(ctx, chainA, userA, denomA, clientIDA, authorityA.String(), 10, 0, 1)

		rateLimit := s.rateLimit(ctx, chainA, denomA, clientIDA)
		s.Require().NotNil(rateLimit)
		s.Require().Equal(int64(10), rateLimit.Quota.MaxPercentSend.Int64())
		s.Require().Equal(int64(0), rateLimit.Quota.MaxPercentRecv.Int64())
		s.Require().Zero(rateLimit.Flow.Outflow.Int64())
	})

	t.Run("ibc v2 transfer updates outflow", func(t *testing.T) {
		balanceBefore, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		packet = s.sendTransferIBCV2(ctx, chainA, userA, clientIDA, denomA, testvalues.IBCTransferAmount, userB.FormattedAddress())

		rateLimit := s.rateLimit(ctx, chainA, denomA, clientIDA)
		s.Require().NotNil(rateLimit)
		s.Require().Equal(testvalues.IBCTransferAmount, rateLimit.Flow.Outflow.Int64())

		balanceAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		s.Require().Equal(balanceBefore-testvalues.IBCTransferAmount, balanceAfter)
	})

	t.Run("self relay v2 packet", func(t *testing.T) {
		ack = s.recvPacketIBCV2(ctx, chainA, chainB, chainACosmos, rlyWalletB, packet)
		s.Require().True(ack.Success())

		s.acknowledgePacketIBCV2(ctx, chainB, chainA, chainBCosmos, rlyWalletA, packet, ack)
	})

	t.Run("verify tokens received on chainB", func(t *testing.T) {
		balance, err := query.Balance(ctx, chainB, userB.FormattedAddress(), ibcTokenB.IBCDenom())
		s.Require().NoError(err)
		s.Require().Equal(testvalues.IBCTransferAmount, balance.Int64())
	})

	t.Run("add incoming v2 rate limit on chainB", func(t *testing.T) {
		s.addRateLimit(ctx, chainB, userB, ibcTokenB.IBCDenom(), clientIDB, authorityB.String(), 100, 0, 1)

		rateLimit := s.rateLimit(ctx, chainB, ibcTokenB.IBCDenom(), clientIDB)
		s.Require().NotNil(rateLimit)
		s.Require().Equal(int64(0), rateLimit.Quota.MaxPercentRecv.Int64())
		s.Require().Zero(rateLimit.Flow.Inflow.Int64())
	})

	t.Run("recv quota denial writes error ack and source ack refunds", func(t *testing.T) {
		outflowBefore := s.rateLimit(ctx, chainA, denomA, clientIDA).Flow.Outflow.Int64()
		balanceBefore, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		packet = s.sendTransferIBCV2(ctx, chainA, userA, clientIDA, denomA, testvalues.IBCTransferAmount, userB.FormattedAddress())
		s.Require().Equal(outflowBefore+testvalues.IBCTransferAmount, s.rateLimit(ctx, chainA, denomA, clientIDA).Flow.Outflow.Int64())

		ack = s.recvPacketIBCV2(ctx, chainA, chainB, chainACosmos, rlyWalletB, packet)
		s.Require().False(ack.Success())
		s.Require().Len(ack.AppAcknowledgements, 1)
		s.Require().Equal(channeltypesv2.ErrorAcknowledgement[:], ack.AppAcknowledgements[0])

		chainBRateLimit := s.rateLimit(ctx, chainB, ibcTokenB.IBCDenom(), clientIDB)
		s.Require().NotNil(chainBRateLimit)
		s.Require().Zero(chainBRateLimit.Flow.Inflow.Int64())

		s.acknowledgePacketIBCV2(ctx, chainB, chainA, chainBCosmos, rlyWalletA, packet, ack)

		s.Require().Equal(outflowBefore, s.rateLimit(ctx, chainA, denomA, clientIDA).Flow.Outflow.Int64())
		balanceAfter, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)
		s.Require().Equal(balanceBefore, balanceAfter)
	})

	t.Run("set outflow quota to 0: ibc v2 transfer fails", func(t *testing.T) {
		s.updateRateLimit(ctx, chainA, userA, denomA, clientIDA, authorityA.String(), 0, 1)

		msgSend := s.newMsgSendPacketIBCV2(userA, clientIDA, denomA, testvalues.IBCTransferAmount, userB.FormattedAddress())
		txResp := s.BroadcastMessages(ctx, chainA, userA, msgSend)
		s.AssertTxFailure(txResp, ratelimitingtypes.ErrQuotaExceeded)
	})
}

func (s *RateLimV2TestSuite) rateLimit(ctx context.Context, chain ibc.Chain, denom, clientID string) *ratelimitingtypes.RateLimit {
	respRateLim, err := query.GRPCQuery[ratelimitingtypes.QueryRateLimitResponse](ctx, chain, &ratelimitingtypes.QueryRateLimitRequest{
		Denom:             denom,
		ChannelOrClientId: clientID,
	})
	s.Require().NoError(err)
	return respRateLim.RateLimit
}

func (s *RateLimV2TestSuite) addRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, clientID, authority string, sendPercent, recvPercent, duration int64) {
	msg := &ratelimitingtypes.MsgAddRateLimit{
		Signer:            authority,
		Denom:             denom,
		ChannelOrClientId: clientID,
		MaxPercentSend:    sdkmath.NewInt(sendPercent),
		MaxPercentRecv:    sdkmath.NewInt(recvPercent),
		DurationHours:     uint64(duration),
	}
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, user)
}

func (s *RateLimV2TestSuite) updateRateLimit(ctx context.Context, chain ibc.Chain, user ibc.Wallet, denom, clientID, authority string, sendPercent, recvPercent int64) {
	msg := &ratelimitingtypes.MsgUpdateRateLimit{
		Signer:            authority,
		Denom:             denom,
		ChannelOrClientId: clientID,
		MaxPercentSend:    sdkmath.NewInt(sendPercent),
		MaxPercentRecv:    sdkmath.NewInt(recvPercent),
		DurationHours:     1,
	}
	s.ExecuteAndPassGovV1Proposal(ctx, msg, chain, user)
}

func (s *RateLimV2TestSuite) sendTransferIBCV2(ctx context.Context, chain ibc.Chain, user ibc.Wallet, clientID, denom string, amount int64, receiver string) channeltypesv2.Packet {
	msgSend := s.newMsgSendPacketIBCV2(user, clientID, denom, amount, receiver)
	txResp := s.BroadcastMessages(ctx, chain, user, msgSend)
	s.AssertTxSuccess(txResp)

	packet, err := ibctesting.ParseV2PacketFromEvents(txResp.Events)
	s.Require().NoError(err)
	s.Require().NotNil(packet)

	return packet
}

func (*RateLimV2TestSuite) newMsgSendPacketIBCV2(user ibc.Wallet, clientID, denom string, amount int64, receiver string) *channeltypesv2.MsgSendPacket {
	token := transfertypes.Token{
		Denom:  transfertypes.NewDenom(denom),
		Amount: strconv.FormatInt(amount, 10),
	}
	packetData := transfertypes.NewFungibleTokenPacketData(
		token.Denom.Path(),
		token.Amount,
		user.FormattedAddress(),
		receiver,
		"",
	)
	payload := channeltypesv2.NewPayload(
		transfertypes.PortID, transfertypes.PortID,
		transfertypes.V1, transfertypes.EncodingJSON, packetData.GetBytes(),
	)

	timeoutTimestamp := uint64(time.Now().Add(10 * time.Minute).Unix())
	return channeltypesv2.NewMsgSendPacket(clientID, timeoutTimestamp, user.FormattedAddress(), payload)
}

func (s *RateLimV2TestSuite) recvPacketIBCV2(ctx context.Context, sourceChain, destinationChain ibc.Chain, sourceChainCosmos *cosmos.CosmosChain, relayer ibc.Wallet, packet channeltypesv2.Packet) channeltypesv2.Acknowledgement {
	targetHeight := s.manualRelayTargetHeight(ctx, sourceChainCosmos, destinationChain, packet.DestinationClient)
	s.updateTendermintClient(ctx, destinationChain, sourceChain, packet.DestinationClient, relayer, targetHeight)

	proof, proofHeight := s.queryPacketCommitmentProof(ctx, sourceChain, packet.SourceClient, packet.Sequence, targetHeight)
	s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

	msgRecv := channeltypesv2.NewMsgRecvPacket(packet, proof, proofHeight, relayer.FormattedAddress())
	txResp := s.BroadcastMessages(ctx, destinationChain, relayer, msgRecv)
	s.AssertTxSuccess(txResp)

	ackBz, err := ibctesting.ParseAckV2FromEvents(txResp.Events)
	s.Require().NoError(err)
	s.Require().NotNil(ackBz)

	var ack channeltypesv2.Acknowledgement
	s.Require().NoError(proto.Unmarshal(ackBz, &ack))
	return ack
}

func (s *RateLimV2TestSuite) acknowledgePacketIBCV2(ctx context.Context, sourceChain, destinationChain ibc.Chain, sourceChainCosmos *cosmos.CosmosChain, relayer ibc.Wallet, packet channeltypesv2.Packet, ack channeltypesv2.Acknowledgement) {
	targetHeight := s.manualRelayTargetHeight(ctx, sourceChainCosmos, destinationChain, packet.SourceClient)
	s.updateTendermintClient(ctx, destinationChain, sourceChain, packet.SourceClient, relayer, targetHeight)

	proof, proofHeight := s.queryPacketAcknowledgementProof(ctx, sourceChain, packet.DestinationClient, packet.Sequence, targetHeight)
	s.Require().Equal(uint64(targetHeight), proofHeight.GetRevisionHeight())

	msgAck := channeltypesv2.NewMsgAcknowledgement(packet, ack, proof, proofHeight, relayer.FormattedAddress())
	txResp := s.BroadcastMessages(ctx, destinationChain, relayer, msgAck)
	s.AssertTxSuccess(txResp)
}

func (s *RateLimV2TestSuite) registerCounterparties(ctx context.Context, chainA, chainB ibc.Chain, rlyWalletA, rlyWalletB ibc.Wallet, clientIDA, clientIDB string) {
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
}

func (s *RateLimV2TestSuite) createTendermintClient(ctx context.Context, hostingChain, counterparty ibc.Chain, signer ibc.Wallet) string {
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

func (s *RateLimV2TestSuite) queryPacketCommitmentProof(ctx context.Context, chain ibc.Chain, clientID string, sequence uint64, targetHeight int64) ([]byte, clienttypes.Height) {
	proofKey := hostv2.PacketCommitmentKey(clientID, sequence)
	proof, proofHeight, err := s.queryProofForIBCStore(ctx, chain, proofKey, targetHeight)
	s.Require().NoError(err)
	return proof, proofHeight
}

func (s *RateLimV2TestSuite) queryPacketAcknowledgementProof(ctx context.Context, chain ibc.Chain, clientID string, sequence uint64, targetHeight int64) ([]byte, clienttypes.Height) {
	proofKey := hostv2.PacketAcknowledgementKey(clientID, sequence)
	proof, proofHeight, err := s.queryProofForIBCStore(ctx, chain, proofKey, targetHeight)
	s.Require().NoError(err)
	return proof, proofHeight
}

func (*RateLimV2TestSuite) queryProofForIBCStore(ctx context.Context, chain ibc.Chain, key []byte, targetHeight int64) ([]byte, clienttypes.Height, error) {
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

func (s *RateLimV2TestSuite) updateTendermintClient(ctx context.Context, hostingChain, counterparty ibc.Chain, clientID string, signer ibc.Wallet, targetHeight int64) {
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

func (s *RateLimV2TestSuite) manualRelayTargetHeight(ctx context.Context, sourceChain *cosmos.CosmosChain, hostingChain ibc.Chain, clientID string) int64 {
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

func (s *RateLimV2TestSuite) queryValidatorSet(ctx context.Context, chain *cosmos.CosmosChain, height int64) *cmttypes.ValidatorSet {
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
