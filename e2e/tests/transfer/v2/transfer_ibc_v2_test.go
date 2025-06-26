//go:build !test_e2e

package v2

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/config"
	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/container"
	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/dockerutil"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypesv2 "github.com/cosmos/ibc-go/v10/modules/core/02-client/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
)

type IBCV2TransferTestSuite struct {
	testsuite.E2ETestSuite
}

func TestIBCV2TransferTestSuite(t *testing.T) {
	suite.Run(t, new(IBCV2TransferTestSuite))
}

func (s *IBCV2TransferTestSuite) SetupSuite() {
	s.SetupChains(context.TODO(), 2, nil)
}

func (s *IBCV2TransferTestSuite) TestIBCV2Transfer() {
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

	// Initiate transfer from chainA to chainB
	timeoutTimestamp := uint64(time.Now().Add(5 * time.Minute).Unix())
	msg := &transfertypes.MsgTransfer{
		SourcePort:       transfertypes.PortID,
		SourceChannel:    firstClientID,
		Token:            testvalues.DefaultTransferAmount(chainA.Config().Denom),
		Sender:           userA.FormattedAddress(),
		Receiver:         userB.FormattedAddress(),
		TimeoutTimestamp: timeoutTimestamp,
		Memo:             "",
	}
	transferOnAResp := s.BroadcastMessages(ctx, chainA, userA, msg)
	s.AssertTxSuccess(transferOnAResp)
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)

	// Verify tokens are escrowed on chainA
	chainABalance, err := s.GetChainANativeBalance(ctx, userA)
	s.Require().NoError(err)
	expectedChainABalance := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
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
	ibcDenom := transfertypes.NewDenom(chainA.Config().Denom, transfertypes.NewHop(transfertypes.PortID, firstClientID)).IBCDenom()
	chainBBalanceResp, err := query.GRPCQuery[banktypes.QueryBalanceResponse](ctx, chainB, &banktypes.QueryBalanceRequest{
		Address: userB.FormattedAddress(),
		Denom:   ibcDenom,
	})
	s.Require().NoError(err)
	s.Require().NotNil(chainBBalanceResp.Balance)
	s.Require().Equal(testvalues.IBCTransferAmount, chainBBalanceResp.Balance.Amount.Int64())

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

// TODO: Move or replace with existing stuff, moved over from solidity-ibc-eureka for convenience for now
func (s *IBCV2TransferTestSuite) BroadcastSdkTxBody(ctx context.Context, chain ibc.Chain, user ibc.Wallet, txBodyBz []byte, retries int) (sdk.TxResponse, error) {
	var txBody txtypes.TxBody
	err := proto.Unmarshal(txBodyBz, &txBody)
	s.Require().NoError(err)

	var msgs []sdk.Msg
	for _, msg := range txBody.Messages {
		var sdkMsg sdk.Msg
		err = chain.Config().EncodingConfig.InterfaceRegistry.UnpackAny(msg, &sdkMsg)
		s.Require().NoError(err)

		msgs = append(msgs, sdkMsg)
	}

	s.Require().NotEmpty(msgs)

	var resp sdk.TxResponse
	for range retries {
		resp, err = s.BroadcastSdkMessages(ctx, chain.(*cosmos.CosmosChain), user, 500_000, msgs...)
		if err == nil && resp.Code == 0 {
			return resp, nil
		}
		if err != nil {
			s.T().Logf("error broadcasting tx: %s", err)
		} else {
			s.T().Logf("tx failed with code %d: %s", resp.Code, resp.RawLog)
		}
		time.Sleep(5 * time.Second)
		s.T().Logf("retrying tx")
	}
	return sdk.TxResponse{}, errors.New("failed to broadcast tx")
}

// TODO: Move or replace with existing stuff, moved over from solidity-ibc-eureka for convenience for now
// BroadcastMessages broadcasts the provided messages to the given chain and signs them on behalf of the provided user.
// Once the broadcast response is returned, we wait for two blocks to be created on chain.
func (s *IBCV2TransferTestSuite) BroadcastSdkMessages(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, gas uint64, msgs ...sdk.Msg) (sdk.TxResponse, error) {
	sdk.GetConfig().SetBech32PrefixForAccount(chain.Config().Bech32Prefix, chain.Config().Bech32Prefix+sdk.PrefixPublic)
	sdk.GetConfig().SetBech32PrefixForValidator(
		chain.Config().Bech32Prefix+sdk.PrefixValidator+sdk.PrefixOperator,
		chain.Config().Bech32Prefix+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic,
	)

	broadcaster := cosmos.NewBroadcaster(s.T(), chain)

	broadcaster.ConfigureClientContextOptions(func(clientContext client.Context) client.Context {
		return clientContext.
			WithCodec(chain.Config().EncodingConfig.Codec).
			WithChainID(chain.Config().ChainID).
			WithTxConfig(chain.Config().EncodingConfig.TxConfig)
	})

	broadcaster.ConfigureFactoryOptions(func(factory tx.Factory) tx.Factory {
		return factory.WithGas(gas)
	})

	resp, err := cosmos.BroadcastTx(ctx, broadcaster, user, msgs...)
	if err != nil {
		return sdk.TxResponse{}, err
	}

	// wait for 2 blocks for the transaction to be included
	s.Require().NoError(testutil.WaitForBlocks(ctx, 2, chain))

	if resp.Code != 0 {
		return sdk.TxResponse{}, fmt.Errorf("tx failed with code %d: %s", resp.Code, resp.RawLog)
	}

	return resp, nil
}
