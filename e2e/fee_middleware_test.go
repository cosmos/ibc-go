package e2e

import (
	"context"
	"github.com/cosmos/ibc-go/v3/e2e/e2efee"
	"github.com/cosmos/ibc-go/v3/e2e/testsuite"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

func TestFeeMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(FeeMiddlewareTestSuite))
}

type FeeMiddlewareTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *FeeMiddlewareTestSuite) TestFeeMiddlewareAsyncMultipleSenders() {
	t := s.T()
	ctx := context.TODO()
	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	srcChain, dstChain := s.GetChains()

	relayer, srcChainChannelInfo := s.CreateRelayerAndChannel(ctx, req, eRep, e2efee.FeeMiddlewareChannelOptions())

	startingTokenAmount := int64(10_000_000)

	srcChainSenderOne := s.CreateUserOnSourceChain(ctx, startingTokenAmount)
	srcChainSenderTwo := s.CreateUserOnSourceChain(ctx, startingTokenAmount)
	dstChainWallet := s.CreateUserOnDestinationChain(ctx, startingTokenAmount)

	t.Run("Relayer wallets can be recovered", func(t *testing.T) {
		req.NoError(s.RecoverRelayerWallets(ctx, relayer))
	})

	srcRelayerWallet, dstRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("Relayer wallets can be fetched", func(t *testing.T) {
		req.NoError(err)
	})

	req.NoError(test.WaitForBlocks(ctx, 10, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Register Counter Party Payee", func(t *testing.T) {
		req.NoError(e2efee.RegisterCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, srcRelayerWallet.Address, srcChainChannelInfo.Counterparty.PortID, srcChainChannelInfo.Counterparty.ChannelID))
		// give some time for update
		time.Sleep(time.Second * 5)
	})

	t.Run("Verify Counter Party Payee", func(t *testing.T) {
		address, err := e2efee.QueryCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, srcChainChannelInfo.Counterparty.ChannelID)
		req.NoError(err)
		req.Equal(srcRelayerWallet.Address, address)
	})

	chain1WalletToChain2WalletAmount := ibc.WalletAmount{
		Address: dstChainWallet.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
		Denom:   srcChain.Config().Denom,
		Amount:  10000,
	}

	var srcTx ibc.Tx
	t.Run("Send IBC transfer", func(t *testing.T) {
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, srcChainChannelInfo.ChannelID, srcChainSenderOne.KeyName, chain1WalletToChain2WalletAmount, nil)
		req.NoError(err)
		req.NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
	})

	t.Run("Verify tokens have been escrowed", func(t *testing.T) {
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainSenderOne)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		req.Equal(expected, actualBalance)
	})

	recvFee := int64(50)
	ackFee := int64(25)
	timeoutFee := int64(10)

	t.Run("Pay packet fee", func(t *testing.T) {
		t.Run("Before paying packet fee there should be no incentivized packets", func(t *testing.T) {
			packets, err := e2efee.QueryPackets(ctx, srcChain, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID)
			req.NoError(err)
			req.Len(packets.IncentivizedPackets, 0)
		})

		t.Run("Paying packet fee should succeed", func(t *testing.T) {
			req.NoError(e2efee.PayPacketFee(ctx, srcChain, srcChainSenderOne.KeyName, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID, 1, recvFee, ackFee, timeoutFee))
			// wait so that incentivised packets will show up
			time.Sleep(5 * time.Second)
		})

		t.Run("Paying packet fee with second sender should succeed", func(t *testing.T) {
			req.NoError(e2efee.PayPacketFee(ctx, srcChain, srcChainSenderTwo.KeyName, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID, 1, recvFee, ackFee, timeoutFee))
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

	t.Run("Balance from first sender should be lowered by sum of recv ack and timeout and IBC transfer amount", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainSenderOne)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - recvFee - ackFee - timeoutFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Balance from second sender should be lowered by sum of recv ack and timeout (not IBC transfer amount)", func(t *testing.T) {
		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainSenderTwo)
		req.NoError(err)

		expected := startingTokenAmount - recvFee - ackFee - timeoutFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Packets should have been relayed", func(t *testing.T) {
		packets, err := e2efee.QueryPackets(ctx, srcChain, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID)
		req.NoError(err)
		req.Len(packets.IncentivizedPackets, 0)
	})

	t.Run("Verify timeout fee is refunded on successful relay of packets for first sender", func(t *testing.T) {
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainSenderOne)
		req.NoError(err)

		gasFee := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFee - ackFee - recvFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Verify timeout fee is refunded on successful relay of packets for second sender", func(t *testing.T) {
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainSenderTwo)
		req.NoError(err)

		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - ackFee - recvFee
		req.Equal(expected, actualBalance)
	})
}

func (s *FeeMiddlewareTestSuite) TestFeeMiddlewareAsyncSingleSender() {
	t := s.T()
	ctx := context.TODO()
	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	srcChain, dstChain := s.GetChains()

	relayer, srcChainChannelInfo := s.CreateRelayerAndChannel(ctx, req, eRep, e2efee.FeeMiddlewareChannelOptions())

	startingTokenAmount := int64(10_000_000)

	srcChainWallet := s.CreateUserOnSourceChain(ctx, startingTokenAmount)
	dstChainWallet := s.CreateUserOnDestinationChain(ctx, startingTokenAmount)

	t.Run("Relayer wallets can be recovered", func(t *testing.T) {
		req.NoError(s.RecoverRelayerWallets(ctx, relayer))
	})

	srcRelayerWallet, dstRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("Relayer wallets can be fetched", func(t *testing.T) {
		req.NoError(err)
	})

	req.NoError(test.WaitForBlocks(ctx, 10, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Register Counter Party Payee", func(t *testing.T) {
		req.NoError(e2efee.RegisterCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, srcRelayerWallet.Address, srcChainChannelInfo.Counterparty.PortID, srcChainChannelInfo.Counterparty.ChannelID))
		// give some time for update
		time.Sleep(time.Second * 5)
	})

	t.Run("Verify Counter Party Payee", func(t *testing.T) {
		address, err := e2efee.QueryCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, srcChainChannelInfo.Counterparty.ChannelID)
		req.NoError(err)
		req.Equal(srcRelayerWallet.Address, address)
	})

	chain1WalletToChain2WalletAmount := ibc.WalletAmount{
		Address: dstChainWallet.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
		Denom:   srcChain.Config().Denom,
		Amount:  10000,
	}

	var srcTx ibc.Tx
	t.Run("Send IBC transfer", func(t *testing.T) {
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, srcChainChannelInfo.ChannelID, srcChainWallet.KeyName, chain1WalletToChain2WalletAmount, nil)
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
			packets, err := e2efee.QueryPackets(ctx, srcChain, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID)
			req.NoError(err)
			req.Len(packets.IncentivizedPackets, 0)
		})

		t.Run("Paying packet fee should succeed", func(t *testing.T) {
			req.NoError(e2efee.PayPacketFee(ctx, srcChain, srcChainWallet.KeyName, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID, 1, recvFee, ackFee, timeoutFee))

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
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainWallet)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - recvFee - ackFee - timeoutFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Packets should have been relayed", func(t *testing.T) {
		packets, err := e2efee.QueryPackets(ctx, srcChain, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID)
		req.NoError(err)
		req.Len(packets.IncentivizedPackets, 0)
	})

	t.Run("Verify timeout fee is refunded on successful relay of packets", func(t *testing.T) {

		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainWallet)
		req.NoError(err)

		gasFee := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - gasFee - ackFee - recvFee
		req.Equal(expected, actualBalance)
	})
}

func (s *FeeMiddlewareTestSuite) TestFeeMiddlewareAsyncSingleSenderTimesOut() {
	t := s.T()
	ctx := context.TODO()
	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	srcChain, dstChain := s.GetChains()

	relayer, srcChainChannelInfo := s.CreateRelayerAndChannel(ctx, req, eRep, e2efee.FeeMiddlewareChannelOptions())

	startingTokenAmount := int64(10_000_000)

	srcChainWallet := s.CreateUserOnSourceChain(ctx, startingTokenAmount)
	dstChainWallet := s.CreateUserOnDestinationChain(ctx, startingTokenAmount)

	t.Run("Relayer wallets can be recovered", func(t *testing.T) {
		req.NoError(s.RecoverRelayerWallets(ctx, relayer))
	})

	srcRelayerWallet, dstRelayerWallet, err := s.GetRelayerWallets(relayer)
	t.Run("Relayer wallets can be fetched", func(t *testing.T) {
		req.NoError(err)
	})

	req.NoError(test.WaitForBlocks(ctx, 10, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Register Counter Party Payee", func(t *testing.T) {
		req.NoError(e2efee.RegisterCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, srcRelayerWallet.Address, srcChainChannelInfo.Counterparty.PortID, srcChainChannelInfo.Counterparty.ChannelID))
		// give some time for update
		time.Sleep(time.Second * 5)
	})

	t.Run("Verify Counter Party Payee", func(t *testing.T) {
		address, err := e2efee.QueryCounterPartyPayee(ctx, dstChain, dstRelayerWallet.Address, srcChainChannelInfo.Counterparty.ChannelID)
		req.NoError(err)
		req.Equal(srcRelayerWallet.Address, address)
	})

	chain1WalletToChain2WalletAmount := ibc.WalletAmount{
		Address: dstChainWallet.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
		Denom:   srcChain.Config().Denom,
		Amount:  10000,
	}

	var srcTx ibc.Tx
	t.Run("Send IBC transfer", func(t *testing.T) {
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, srcChainChannelInfo.ChannelID, srcChainWallet.KeyName, chain1WalletToChain2WalletAmount, &ibc.IBCTimeout{
			NanoSeconds: 100, // want it to timeout immediately
		})
		req.NoError(err)
		req.NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
		time.Sleep(1 * time.Second) // cause timeout
	})

	t.Run("Verify tokens have been escrowed (relayer has not yet picked up the packet)", func(t *testing.T) {
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainWallet)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		req.Equal(expected, actualBalance)
	})

	recvFee := int64(50)
	ackFee := int64(25)
	timeoutFee := int64(10)

	t.Run("Pay packet fee", func(t *testing.T) {
		t.Run("Before paying packet fee there should be no incentivized packets", func(t *testing.T) {
			packets, err := e2efee.QueryPackets(ctx, srcChain, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID)
			req.NoError(err)
			req.Len(packets.IncentivizedPackets, 0)
		})

		t.Run("Paying packet fee should succeed", func(t *testing.T) {
			req.NoError(e2efee.PayPacketFee(ctx, srcChain, srcChainWallet.KeyName, srcChainChannelInfo.PortID, srcChainChannelInfo.ChannelID, 1, recvFee, ackFee, timeoutFee))

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
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainWallet)
		req.NoError(err)

		expected := startingTokenAmount - chain1WalletToChain2WalletAmount.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - recvFee - ackFee - timeoutFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Start relayer", func(t *testing.T) {
		s.StartRelayer(relayer)
	})

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Funds recv and ack should be refunded as the packet timed out", func(t *testing.T) {
		actualBalance, err := s.GetSourceChainBalance(ctx, srcChainWallet)
		req.NoError(err)

		expected := startingTokenAmount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - timeoutFee
		req.Equal(expected, actualBalance)
	})
}
