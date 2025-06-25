//go:build !test_e2e

package v2

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/config"
	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/container"
	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/dockerutil"
	testifysuite "github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	sdkmath "cosmossdk.io/math"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

func TestMultiPayloadTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MultiPayloadTestSuite))
}

type MultiPayloadTestSuite struct {
	IBCV2TransferTestSuite
}

// SetupSuite sets up chains for the current test suite
func (s *MultiPayloadTestSuite) SetupTest() {
	s.SetupChains(context.TODO(), 2, nil)
}

func (s *MultiPayloadTestSuite) TestMultiPayload() {
	t := s.T()
	ctx := context.TODO()
	logger := zap.NewExample()

	chainA, chainB := s.GetChains()
	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	relayerUserA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	relayerUserB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	firstClientID := "07-tendermint-0"

	// Set up docker util
	docker, err := dockerutil.DockerWithExistingSetup(ctx, logger, t.Name(), s.DockerClient, s.Network)
	s.Require().NoError(err)

	// Spin up relayer API
	relayerConfig := config.NewConfig(config.CreateCosmosCosmosModules(config.CosmosToCosmosConfigInfo{
		ChainAID:    chainA.Config().ChainID,
		ChainBID:    chainB.Config().ChainID,
		ChainATmRPC: chainA.GetRPCAddress(),
		ChainBTmRPC: chainB.GetRPCAddress(),
		ChainAUser:  relayerUserA.FormattedAddress(),
		ChainBUser:  relayerUserB.FormattedAddress(),
	}))
	relayerConfig.Address = "0.0.0.0"
	relayerAPIContainer, err := container.SpinUpRelayerApiContainer(ctx, logger, docker, "v0.6.0", relayerConfig, []string{"v1.2.0"})
	s.Require().NoError(err)
	t.Cleanup(func() {
		ctx := context.TODO()
		if err := docker.Cleanup(ctx, true); err != nil {
			t.Logf("error cleaning up docker: %s", err)
		}
	})

	// Create light clients on chainA and chainB
	createClientTxOnA, err := relayerAPIContainer.GetCreateClientTx(ctx, chainB.Config().ChainID, chainA.Config().ChainID)
	s.Require().NoError(err)
	createClientOnAResp, err := s.BroadcastSdkTxBody(ctx, chainA, relayerUserA, createClientTxOnA, 5)
	s.Require().NoError(err)
	s.AssertTxSuccess(createClientOnAResp)
	createClientTxOnB, err := relayerAPIContainer.GetCreateClientTx(ctx, chainA.Config().ChainID, chainB.Config().ChainID)
	s.Require().NoError(err)
	createClientOnBResp, err := s.BroadcastSdkTxBody(ctx, chainB, relayerUserB, createClientTxOnB, 5)
	s.Require().NoError(err)
	s.AssertTxSuccess(createClientOnBResp)

	// Register counterparty client on chainA and chainB
	merklePathPrefix := [][]byte{[]byte(ibcexported.StoreKey), []byte("")}
	registerCounterpartyOnAResp := s.BroadcastMessages(ctx, chainA, relayerUserA, &clienttypesv2.MsgRegisterCounterparty{
		ClientId:                 firstClientID,
		CounterpartyClientId:     firstClientID,
		CounterpartyMerklePrefix: merklePathPrefix,
		Signer:                   relayerUserA.FormattedAddress(),
	})
	s.AssertTxSuccess(registerCounterpartyOnAResp)
	registerCounterpartyOnBResp := s.BroadcastMessages(ctx, chainB, relayerUserB, &clienttypesv2.MsgRegisterCounterparty{
		ClientId:                 firstClientID,
		CounterpartyClientId:     firstClientID,
		CounterpartyMerklePrefix: merklePathPrefix,
		Signer:                   relayerUserB.FormattedAddress(),
	})
	s.AssertTxSuccess(registerCounterpartyOnBResp)

	// Initiate transfer multipayload packet from chainA to chainB
	timeoutTimestamp := uint64(time.Now().Add(5 * time.Minute).Unix())
	data := transfertypes.NewFungibleTokenPacketData(
		chainA.Config().Denom,
		sdkmath.NewInt(testvalues.IBCTransferAmount).String(),
		userA.FormattedAddress(),
		userB.FormattedAddress(),
		"",
	)
	bz, err := transfertypes.MarshalPacketData(data, transfertypes.V1, transfertypes.EncodingJSON)
	s.Require().NoError(err)
	payload := channeltypesv2.NewPayload(
		transfertypes.PortID, transfertypes.PortID,
		transfertypes.V1, transfertypes.EncodingJSON,
		bz,
	)

	// create the MsgSendPacket with two payloads
	msg := channeltypesv2.NewMsgSendPacket(
		firstClientID, timeoutTimestamp,
		userA.FormattedAddress(), payload, payload,
	)
	transferOnAResp := s.BroadcastMessages(ctx, chainA, userA, msg)
	s.AssertTxSuccess(transferOnAResp)
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)

	// Verify tokens are escrowed on chainA, this should be twice the transfer amount
	// because we are sending two payloads in the same packet
	chainABalance, err := s.GetChainANativeBalance(ctx, userA)
	s.Require().NoError(err)
	expectedChainABalance := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount*2
	s.Require().Equal(expectedChainABalance, chainABalance)

	// Relay the packet from chainA to chainB
	transferOnARespTxHash, err := hex.DecodeString(transferOnAResp.TxHash)
	s.Require().NoError(err)
	receiveTx, err := relayerAPIContainer.GetRelayTx(ctx, chainA.Config().ChainID, chainB.Config().ChainID, firstClientID, firstClientID, transferOnARespTxHash)
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)
	receiveOnBResp, err := s.BroadcastSdkTxBody(ctx, chainB, relayerUserB, receiveTx, 5)
	s.Require().NoError(err)
	s.AssertTxSuccess(receiveOnBResp)

	// Verify tokens are transferred to chainB
	// this should be twice the transfer amount
	// because we are sending two payloads in the same packet
	ibcDenom := transfertypes.NewDenom(chainA.Config().Denom, transfertypes.NewHop(transfertypes.PortID, firstClientID)).IBCDenom()
	chainBBalanceResp, err := query.GRPCQuery[banktypes.QueryBalanceResponse](ctx, chainB, &banktypes.QueryBalanceRequest{
		Address: userB.FormattedAddress(),
		Denom:   ibcDenom,
	})
	s.Require().NoError(err)
	s.Require().NotNil(chainBBalanceResp.Balance)
	s.Require().Equal(testvalues.IBCTransferAmount*2, chainBBalanceResp.Balance.Amount.Int64())

	// Relay ack back from chainB to chainA
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)
	relayOnBRespTxHash, err := hex.DecodeString(receiveOnBResp.TxHash)
	s.Require().NoError(err)
	relayAckTx, err := relayerAPIContainer.GetRelayTx(ctx, chainB.Config().ChainID, chainA.Config().ChainID, firstClientID, firstClientID, relayOnBRespTxHash)
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)
	relayAckOnAResp, err := s.BroadcastSdkTxBody(ctx, chainA, relayerUserA, relayAckTx, 5)
	s.Require().NoError(err)
	s.AssertTxSuccess(relayAckOnAResp)
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)

	// Verify packet commitment is deleted (i.e. packet is acknowledged) on chainA
	_, err = query.GRPCQuery[channeltypesv2.QueryPacketCommitmentResponse](ctx, chainA, &channeltypesv2.QueryPacketCommitmentRequest{
		ClientId: firstClientID,
		Sequence: 1,
	})
	s.Require().ErrorContains(err, "packet commitment hash not found")
}
