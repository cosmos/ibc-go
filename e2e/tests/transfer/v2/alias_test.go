//go:build !test_e2e

package v2

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/cosmos/ibc-go/e2e/testsuite/query"
	"github.com/cosmos/ibc-go/e2e/testvalues"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"

	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"

	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/config"
	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/container"
	"github.com/cosmos/solidity-ibc-eureka/packages/go-relayer-api/dockerutil"

	"go.uber.org/zap"
)

func TestAliasTestSuite(t *testing.T) {
	testifysuite.Run(t, new(AliasTestSuite))
}

type AliasTestSuite struct {
	IBCV2TransferTestSuite
}

// SetupSuite sets up chains for the current test suite
func (s *AliasTestSuite) SetupTest() {
	s.SetupChains(context.TODO(), 2, nil)
}

func (s *AliasTestSuite) TestAlias() {
	t := s.T()
	ctx := context.TODO()
	logger := zap.NewExample()

	testName := t.Name()

	// NOTE: t.Parallel() should be called before SetupPath in all tests.
	// t.Name() must be stored in a variable before t.Parallel() otherwise t.Name() is not
	// deterministic.
	t.Parallel()

	chainA, chainB := s.GetChains()
	userA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	userAAddress := userA.FormattedAddress()
	userB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)
	userBAddress := userB.FormattedAddress()
	relayerUserA := s.CreateUserOnChainA(ctx, testvalues.StartingTokenAmount)
	relayerUserB := s.CreateUserOnChainB(ctx, testvalues.StartingTokenAmount)

	// Create a V1 path between chain A and chain B
	s.CreatePaths(ibc.DefaultClientOpts(), s.TransferChannelOptions(), testName)
	v1Relayer := s.GetRelayerForTest(testName)
	channelA := s.GetChannelBetweenChains(testName, chainA, chainB)

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

	/// STAGE 1: Initiate transfer from chainA to chainB
	// with IBC V2 protocol using the v1 channel ID

	timeoutTimestamp := uint64(time.Now().Add(5 * time.Minute).Unix())
	msg := &transfertypes.MsgTransfer{
		SourcePort:       transfertypes.PortID,
		SourceChannel:    channelA.ChannelID,
		Token:            testvalues.DefaultTransferAmount(chainA.Config().Denom),
		Sender:           userA.FormattedAddress(),
		Receiver:         userB.FormattedAddress(),
		TimeoutTimestamp: timeoutTimestamp,
		Memo:             "",
		UseAliasing:      true, // Enable aliasing for the transfer
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
	// with IBC V2 protocol using the v1 channel IDs
	transferOnARespTxHash, err := hex.DecodeString(transferOnAResp.TxHash)
	s.Require().NoError(err)
	receiveTx, err := relayerAPIContainer.GetRelayTx(ctx, chainA.Config().ChainID, chainB.Config().ChainID, channelA.ChannelID, channelA.Counterparty.ChannelID, transferOnARespTxHash)
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)
	receiveOnBResp, err := s.BroadcastSdkTxBody(ctx, chainB, relayerUserB, receiveTx, 5)
	s.Require().NoError(err)
	s.AssertTxSuccess(receiveOnBResp)

	// Verify tokens are transferred to chainB
	ibcDenom := transfertypes.NewDenom(chainA.Config().Denom, transfertypes.NewHop(transfertypes.PortID, channelA.Counterparty.ChannelID)).IBCDenom()
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
	relayAckTx, err := relayerAPIContainer.GetRelayTx(ctx, chainB.Config().ChainID, chainA.Config().ChainID, channelA.Counterparty.ChannelID, channelA.ChannelID, relayOnBRespTxHash)
	s.Require().NoError(err)
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)
	relayAckOnAResp, err := s.BroadcastSdkTxBody(ctx, chainA, relayerUserA, relayAckTx, 5)
	s.Require().NoError(err)
	s.AssertTxSuccess(relayAckOnAResp)
	err = testutil.WaitForBlocks(ctx, 1, chainA, chainB)
	s.Require().NoError(err)

	// Verify packet commitment is deleted (i.e. packet is acknowledged) on chainA
	// again we use the v1 channel ID in place of the client ID in IBC V2 protocol
	_, err = query.GRPCQuery[channeltypesv2.QueryPacketCommitmentResponse](ctx, chainA, &channeltypesv2.QueryPacketCommitmentRequest{
		ClientId: channelA.ChannelID,
		Sequence: 1,
	})
	s.Require().ErrorContains(err, "packet commitment hash not found")

	/// STAGE 2: Use the v1 protocol to send the same tokens back to chain A

	t.Run("IBC token transfer of vouchers from chainB back to chainA, receiver chain is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainB, userB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, testvalues.DefaultTransferAmount(ibcDenom), userBAddress, userAAddress, s.GetTimeoutHeight(ctx, chainA), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("vouchers are burned", func(t *testing.T) {
		actualBalance, err := testsuite.GetChainBalanceForDenom(ctx, chainB, ibcDenom, userB)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		// TODO: cannot query total escrow if tests in parallel are using the same denom.
		// if testvalues.TotalEscrowFeatureReleases.IsSupported(chainAVersion) {
		//	actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
		//	s.Require().NoError(err)
		//
		//	expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
		//	s.Require().Equal(expectedTotalEscrow, actualTotalEscrow)
		// }
	})

	t.Run("start relayer", func(t *testing.T) {
		s.StartRelayer(v1Relayer, testName)
	})

	t.Run("v1 packets are relayed", func(t *testing.T) {
		s.AssertPacketRelayed(ctx, chainB, channelA.Counterparty.PortID, channelA.Counterparty.ChannelID, 1)

		actualBalance, err := query.Balance(ctx, chainA, userAAddress, chainA.Config().Denom)
		s.Require().NoError(err)

		// sender should have the same balance as before the first transfer
		// since we have now sent the tokens back to chain A using IBC v1
		expected := testvalues.StartingTokenAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

	/// STAGE 3: Use the v1 protocol to send the original tokens back to chainB
	// Ensure that the sequence number is incremented correctly

	t.Run("native IBC token transfer from chainA to chainB, sender is source of tokens", func(t *testing.T) {
		transferTxResp := s.Transfer(ctx, chainA, userA, channelA.PortID, channelA.ChannelID, testvalues.DefaultTransferAmount(chainA.Config().Denom), userAAddress, userBAddress, s.GetTimeoutHeight(ctx, chainB), 0, "")
		s.AssertTxSuccess(transferTxResp)
	})

	t.Run("tokens are escrowed", func(t *testing.T) {
		actualBalance, err := s.GetChainANativeBalance(ctx, userA)
		s.Require().NoError(err)

		expected := testvalues.StartingTokenAmount - testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance)

		// TODO: cannot query total escrow if tests in parallel are using the same denom.
		// if testvalues.TotalEscrowFeatureReleases.IsSupported(chainAVersion) {
		//	actualTotalEscrow, err := query.TotalEscrowForDenom(ctx, chainA, chainADenom)
		//	s.Require().NoError(err)
		//
		//	expectedTotalEscrow := sdk.NewCoin(chainADenom, sdkmath.NewInt(testvalues.IBCTransferAmount))
		//	s.Require().Equal(expectedTotalEscrow, actualTotalEscrow)
		// }
	})

	t.Run("packets are relayed", func(t *testing.T) {
		// ensure that the sequence number is incremented correctly
		// since the first packet on this channel was sent using IBC V2 protocol
		s.AssertPacketRelayed(ctx, chainA, channelA.PortID, channelA.ChannelID, 2)

		actualBalance, err := query.Balance(ctx, chainB, userBAddress, ibcDenom)
		s.Require().NoError(err)

		expected := testvalues.IBCTransferAmount
		s.Require().Equal(expected, actualBalance.Int64())
	})

}
