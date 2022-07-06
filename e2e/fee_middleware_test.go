package e2e

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/ibc-go/v4/e2e/testsuite"
	feetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
)

func TestFeeMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *FeeMiddlewareTestSuite) RegisterCounterPartyPayee(ctx context.Context, chain *cosmos.CosmosChain /*user  broadcast.User*/, portID, channelID, relayerAddr, counterpartyPayeeAddr string) (sdk.TxResponse, error) {
	_ = feetypes.NewMsgRegisterCounterpartyPayee(portID, channelID, relayerAddr, counterpartyPayeeAddr)
	return sdk.TxResponse{}, nil
}

func (s *FeeMiddlewareTestSuite) QueryCounterPartyPayee(ctx context.Context, chain ibc.Chain, relayerAddress, channelID string) (string, error) {
	queryClient := s.GetChainClientSet(chain).FeeQueryClient
	res, err := queryClient.CounterpartyPayee(ctx, &feetypes.QueryCounterpartyPayeeRequest{
		ChannelId: channelID,
		Relayer:   relayerAddress,
	})
	if err != nil {
		return "", err
	}
	return res.CounterpartyPayee, nil
}

func (s *FeeMiddlewareTestSuite) TestAsyncSingleSender() {
	t := s.T()
	ctx := context.TODO()

	relayer := s.CreateChainsRelayerAndChannel(ctx, feeMiddlewareChannelOptions())
	srcChain, dstChain := s.GetChains()

	//startingTokenAmount := int64(10_000_000)

	//srcChainWallet := s.CreateUserOnSourceChain(ctx, startingTokenAmount)
	//dstChainWallet := s.CreateUserOnDestinationChain(ctx, startingTokenAmount)

	t.Run("relayer wallets recovered", func(t *testing.T) {
		s.Require().NoError(s.RecoverRelayerWallets(ctx, relayer))
	})

	srcRelayerWallet, dstRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("relayer wallets fetched", func(t *testing.T) {
		s.Require().NoError(err)
	})

	s.Require().NoError(test.WaitForBlocks(ctx, 10, srcChain, dstChain), "failed to wait for blocks")

	t.Run("register counter party payee", func(t *testing.T) {
		tx, err := s.RegisterCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, srcRelayerWallet.Address, "transfer", "channel-0")
		s.Require().NoError(err)
		s.AssertValidTxResponse(tx)
		// give some time for update
		time.Sleep(time.Second * 5)
	})

	t.Run("verify counter party payee", func(t *testing.T) {
		address, err := s.QueryCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, "channel-0")
		s.Require().NoError(err)
		s.Require().Equal(srcRelayerWallet.Address, address)
	})

	//chain1WalletToChain2WalletAmount := ibc.WalletAmount{
	//	Address: dstChainWallet.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
	//	Denom:   srcChain.Config().Denom,
	//	Amount:  10000,
	//}
	//
	//var srcTx ibc.Tx
	//t.Run("send IBC transfer", func(t *testing.T) {
	//	var err error
	//	srcTx, err = srcChain.SendIBCTransfer(ctx, srcChainChannelInfo.ChannelID, srcChainWallet.KeyName, chain1WalletToChain2WalletAmount, nil)
	//	s.Require().NoError(err)
	//	s.Require().NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
	//})
	//
	//t.Run("tokens are escrowed", func(t *testing.T) {
	//	actualBalance, err := s.GetSourceChainNativeBalance(ctx, srcChainWallet)
	//	s.Require().NoError(err)
	//
	//	expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
	//	s.Require().Equal(expected, actualBalance)
	//})
	//
	//recvFee := int64(50)
	//ackFee := int64(25)
	//timeoutFee := int64(10)
	//
	//t.Run("pay packet fee", func(t *testing.T) {
	//	t.Run("no incentivized packets", s.AssertEmptyPackets(ctx, srcChain, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID))
	//
	//	t.Run("should succeed", func(t *testing.T) {
	//		s.Require().NoError(e2efee.PayPacketFee(ctx, srcChain, srcChainWallet.KeyName, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID, 1, recvFee, ackFee, timeoutFee))
	//
	//		// wait so that incentivised packets will show up
	//		time.Sleep(5 * time.Second)
	//	})
	//
	//	t.Run("there should be incentivized packets", func(t *testing.T) {
	//		packets, err := e2efee.QueryPackets(ctx, srcChain, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID)
	//		s.Require().NoError(err)
	//		s.Require().Len(packets.IncentivizedPackets, 1)
	//		actualFee := packets.IncentivizedPackets[0].PacketFees[0].Fee
	//
	//		expectedRecv, expectedAck, exectedTimeout := convertFeeAmountsToCoins(srcChain.Config().Denom, recvFee, ackFee, timeoutFee)
	//		s.Require().True(actualFee.RecvFee.IsEqual(expectedRecv))
	//		s.Require().True(actualFee.AckFee.IsEqual(expectedAck))
	//		s.Require().True(actualFee.TimeoutFee.IsEqual(exectedTimeout))
	//	})
	//})
	//
	//t.Run("balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
	//	// The balance should be lowered by the sum of the recv, ack and timeout fees.
	//	actualBalance, err := s.GetSourceChainNativeBalance(ctx, srcChainWallet)
	//	s.Require().NoError(err)
	//
	//	expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - recvFee - ackFee - timeoutFee
	//	s.Require().Equal(expected, actualBalance)
	//})
	//
	//t.Run("start relayer", func(t *testing.T) {
	//	s.StartRelayer(relayer)
	//})
	//
	//s.Require().NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")
	//
	//t.Run("packets are relayed", s.AssertEmptyPackets(ctx, srcChain, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID))
	//
	//t.Run("timeout fee is refunded", func(t *testing.T) {
	//
	//	actualBalance, err := s.GetSourceChainNativeBalance(ctx, srcChainWallet)
	//	s.Require().NoError(err)
	//
	//	gasFee := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
	//	// once the relayer has relayed the packets, the timeout fee should be refunded.
	//	expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFee - ackFee - recvFee
	//	s.Require().Equal(expected, actualBalance)
	//})
}

// feeMiddlewareChannelOptions configures both of the chains to have fee middleware enabled.
func feeMiddlewareChannelOptions() func(options *ibc.CreateChannelOptions) {
	return func(opts *ibc.CreateChannelOptions) {
		opts.Version = "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
		opts.DestPortName = "transfer"
		opts.SourcePortName = "transfer"
	}
}
