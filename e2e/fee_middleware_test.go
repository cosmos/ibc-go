package e2e

import (
	"context"
	"github.com/cosmos/ibc-go/v3/e2e/e2efee"
	"github.com/cosmos/ibc-go/v3/e2e/testsuite"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
	"time"
)

func TestFeeMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
	testsuite.E2ETestSuite
	chainPairs map[string]feeMiddlewareChainPair
}

func (s *FeeMiddlewareTestSuite) SetupSuite() {
	s.chainPairs = map[string]feeMiddlewareChainPair{}
}

func (s *FeeMiddlewareTestSuite) SetupTest() {
	srcChain, dstChain := s.CreateCosmosChains()
	s.chainPairs[s.T().Name()] = feeMiddlewareChainPair{
		srcChain: &e2efee.FeeMiddlewareChain{CosmosChain: srcChain},
		dstChain: &e2efee.FeeMiddlewareChain{CosmosChain: dstChain},
	}
}

// feeMiddlewareChainPair holds the source and destination chain.
// for these tests, we need to wrap the cosmos.CosmosChain to provide more functionality.
type feeMiddlewareChainPair struct {
	srcChain, dstChain *e2efee.FeeMiddlewareChain
}

func (s *FeeMiddlewareTestSuite) TestFeeMiddleware() {
	t := s.T()
	ctx := context.TODO()
	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	chainPair, ok := s.chainPairs[t.Name()]
	req.True(ok)

	srcChain := chainPair.srcChain
	dstChain := chainPair.dstChain

	relayer, srcChainChannel, startRelayerFunc := s.CreateRelayerAndChannel(ctx, srcChain, dstChain, req, eRep)

	startingTokenAmount := int64(10_000_000)

	users := ibctest.GetAndFundTestUsers(t, ctx, strings.ReplaceAll(t.Name(), " ", "-"), startingTokenAmount, srcChain, dstChain)

	srcChainWallet := users[0]
	dstChainWallet := users[1]

	srcRelayWallet, ok := relayer.GetWallet(srcChain.Config().ChainID)
	req.True(ok)
	dstRelayWallet, ok := relayer.GetWallet(dstChain.Config().ChainID)
	req.True(ok)

	req.NoError(srcChain.RecoverKeyring(ctx, "rly1", srcRelayWallet.Mnemonic))
	req.NoError(dstChain.RecoverKeyring(ctx, "rly2", dstRelayWallet.Mnemonic))

	srcRelayBal, err := srcChain.GetBalance(ctx, srcRelayWallet.Address, srcChain.Config().Denom)
	req.NoError(err)
	t.Logf("SRC RELAY BAL %d", srcRelayBal)
	req.NotEmpty(srcRelayBal)
	dstRelayBal, err := dstChain.GetBalance(ctx, dstRelayWallet.Address, dstChain.Config().Denom)
	req.NoError(err)
	t.Logf("DST RELAY BAL %d", dstRelayBal)
	req.NotEmpty(dstRelayBal)

	req.NoError(test.WaitForBlocks(ctx, 10, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Register Counter Party Payee", func(t *testing.T) {
		req.NoError(dstChain.RegisterCounterPartyPayee(ctx, dstRelayWallet.Address, srcRelayWallet.Address, srcChainChannel.Counterparty.PortID, srcChainChannel.Counterparty.ChannelID))
		// give some time for update
		time.Sleep(time.Second * 5)
	})

	t.Run("Verify Counter Party Payee", func(t *testing.T) {
		address, err := dstChain.QueryCounterPartyPayee(ctx, dstRelayWallet.Address, srcChainChannel.Counterparty.ChannelID)
		req.NoError(err)
		req.Equal(srcRelayWallet.Address, address)
	})

	chain1WalletToChain2WalletAmount := ibc.WalletAmount{
		Address: dstChainWallet.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
		Denom:   srcChain.Config().Denom,
		Amount:  10000,
	}

	var srcTx ibc.Tx
	t.Run("Send IBC transfer", func(t *testing.T) {
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, srcChainChannel.ChannelID, srcChainWallet.KeyName, chain1WalletToChain2WalletAmount, nil)
		req.NoError(err)
		req.NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
	})

	t.Run("Verify tokens have been escrowed", func(t *testing.T) {
		actualBalance, err := srcChain.GetBalance(ctx, srcChainWallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		req.Equal(expected, actualBalance)
	})

	recvFee := int64(50)
	ackFee := int64(25)
	timeoutFee := int64(10)

	t.Run("Pay packet fee", func(t *testing.T) {
		t.Run("Before paying packet fee there should be no incentivized packets", func(t *testing.T) {
			packets, err := srcChain.QueryPackets(ctx, srcChainChannel.PortID, srcChainChannel.PortID)
			req.NoError(err)
			req.Len(packets.IncentivizedPackets, 0)
		})

		t.Run("Paying packet fee should succeed", func(t *testing.T) {
			err := srcChain.PayPacketFee(ctx, srcChainWallet.KeyName, srcChainChannel.PortID, srcChainChannel.ChannelID, 1, recvFee, ackFee, timeoutFee)
			req.NoError(err)

			// wait so that incentivised packets will show up
			time.Sleep(5 * time.Second)
		})

		// TODO: query method not umarshalling json correctly yet.
		//t.Run("After paying packet fee there should be incentivized packets", func(t *testing.T) {
		//	packets, err := srcChain.QueryPackets(ctx, "transfer", "channel-0")
		//	req.NoError(err)
		//	req.Len(packets.IncentivizedPackets, 1)
		//})
	})

	t.Run("Balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := srcChain.GetBalance(ctx, srcChainWallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - recvFee - ackFee - timeoutFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Start relayer", startRelayerFunc)

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Packets should have been relayed", func(t *testing.T) {
		packets, err := srcChain.QueryPackets(ctx, srcChainChannel.PortID, srcChainChannel.ChannelID)
		req.NoError(err)
		req.Len(packets.IncentivizedPackets, 0)
	})

	t.Run("Verify recv fees are refunded when no forward relayer is found", func(t *testing.T) {
		actualBalance, err := srcChain.GetBalance(ctx, srcChainWallet.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		gasFee := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFee - ackFee - recvFee
		req.Equal(expected, actualBalance)
	})
}
