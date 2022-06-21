package e2e

import (
	"context"
	"fmt"
	"github.com/cosmos/ibc-go/v3/e2e/e2efee"
	"github.com/cosmos/ibc-go/v3/e2e/setup"
	"github.com/cosmos/ibc-go/v3/e2e/testconfig"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestFeeMiddlewareAsync(t *testing.T) {
	ctx := context.TODO()
	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	srcChain, dstChain, relayer := setup.StandardTwoChainEnvironment(t, req, eRep, setup.FeeMiddlewareOptions())

	startingTokenAmount := int64(10_000_000)

	users := ibctest.GetAndFundTestUsers(t, ctx, strings.ReplaceAll(t.Name(), " ", "-"), startingTokenAmount, srcChain, dstChain, srcChain, dstChain)

	// TODO: use real relayer addresses
	srcRelayUser := users[0]
	dstRelayUser := users[1]

	wallet1Chain1 := users[2]
	wallet3Chain2 := users[3]

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	dstFeeChain := &e2efee.FeeMiddlewareChain{CosmosChain: dstChain}
	srcFeeChain := &e2efee.FeeMiddlewareChain{CosmosChain: srcChain}

	t.Run("Register Counter Party Payee", func(t *testing.T) {
		req.NoError(dstFeeChain.RegisterCounterPartyPayee(ctx, srcRelayUser.Bech32Address(srcChain.Config().Bech32Prefix), dstRelayUser.Bech32Address(dstFeeChain.Config().Bech32Prefix)))
	})

	testCoinWallet1ToWallet3 := ibc.WalletAmount{
		Address: wallet3Chain2.Bech32Address(dstChain.Config().Bech32Prefix), // destination address
		Denom:   srcChain.Config().Denom,
		Amount:  10000,
	}

	var srcTx ibc.Tx

	t.Run("Send IBC transfer", func(t *testing.T) {
		// send a transfer from wallet 1 on src chain to wallet 3 on dst chain
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, "channel-0", wallet1Chain1.KeyName, testCoinWallet1ToWallet3, nil)
		req.NoError(err)
		req.NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
	})

	t.Run("Verify tokens have been escrowed", func(t *testing.T) {
		actualBalance, err := srcChain.GetBalance(ctx, wallet1Chain1.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		expected := startingTokenAmount - testCoinWallet1ToWallet3.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		req.Equal(expected, actualBalance)
	})

	recvFee := int64(50)
	ackFee := int64(25)
	timeoutFee := int64(10)

	t.Run("Pay packet fee", func(t *testing.T) {
		err := srcFeeChain.QueryPackets(ctx)
		req.NoError(err)

		err = srcFeeChain.PayPacketFee(ctx, wallet1Chain1.KeyName, recvFee, ackFee, timeoutFee)
		req.NoError(err)

		// wait so that incentivised packets will show up
		time.Sleep(10 * time.Second)
	})

	t.Run("Balance should be lowered by sum of recv ack and timeout", func(t *testing.T) {
		err := srcFeeChain.QueryPackets(ctx)
		req.NoError(err)

		// The balance should be lowered by the sum of the recv, ack and timeout fees.
		actualBalance, err := srcChain.GetBalance(ctx, wallet1Chain1.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		expected := startingTokenAmount - testCoinWallet1ToWallet3.Amount - srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent) - recvFee - ackFee - timeoutFee
		req.Equal(expected, actualBalance)
	})

	t.Run("Start relayer", func(t *testing.T) {
		err := relayer.StartRelayer(ctx, eRep, testconfig.TestPath)
		req.NoError(err, fmt.Sprintf("failed to start relayer: %s", err))
		t.Cleanup(func() {
			if err := relayer.StopRelayer(ctx, eRep); err != nil {
				t.Logf("error stopping relayer: %v", err)
			}
		})
		// wait for relayer to start.
		time.Sleep(time.Second * 10)
	})

	err := srcFeeChain.QueryPackets(ctx)
	req.NoError(err)

	req.NoError(test.WaitForBlocks(ctx, 5, srcChain, dstChain), "failed to wait for blocks")

	t.Run("Verify ack and recv fees are paid and timeout is refunded", func(t *testing.T) {
		actualBalance, err := srcChain.GetBalance(ctx, wallet1Chain1.Bech32Address(srcChain.Config().Bech32Prefix), srcChain.Config().Denom)
		req.NoError(err)

		gasFee := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
		// once the relayer has relayed the packets, the timeout fee should be refunded.
		expected := startingTokenAmount - testCoinWallet1ToWallet3.Amount - gasFee - ackFee - recvFee
		req.Equal(actualBalance, expected)
	})
}
