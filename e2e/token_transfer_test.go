package e2e

import (
	"context"
	"fmt"
	"github.com/cosmos/ibc-go/v3/e2e/setup"
	"github.com/cosmos/ibc-go/v3/e2e/testconfig"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"strings"
	"testing"
	"time"
)

const (
	pollHeightMax = uint64(50)
)

func TestTokenTransfer(t *testing.T) {
	ctx := context.TODO()
	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	srcChain, dstChain, relayer := setup.StandardTwoChainEnvironment(t, req, eRep)

	srcChainCfg := srcChain.Config()
	dstChainCfg := dstChain.Config()

	testUsers := ibctest.GetAndFundTestUsers(t, ctx, strings.ReplaceAll(t.Name(), " ", "-"), 10_000_000, srcChain, dstChain)

	srcUser := testUsers[0]
	dstUser := testUsers[1]

	// will send ibc transfers from user wallet on both chains to their own respective wallet on the other chain
	testCoinSrcToDst := ibc.WalletAmount{
		Address: srcUser.Bech32Address(dstChainCfg.Bech32Prefix),
		Denom:   srcChainCfg.Denom,
		Amount:  10000,
	}
	testCoinDstToSrc := ibc.WalletAmount{
		Address: dstUser.Bech32Address(srcChainCfg.Bech32Prefix),
		Denom:   dstChainCfg.Denom,
		Amount:  20000,
	}

	var (
		eg    errgroup.Group
		srcTx ibc.Tx
		dstTx ibc.Tx
	)

	channelId := "channel-0"
	eg.Go(func() error {
		var err error
		srcTx, err = srcChain.SendIBCTransfer(ctx, channelId, srcUser.KeyName, testCoinSrcToDst, nil)
		if err != nil {
			return fmt.Errorf("failed to send ibc transfer from source: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		var err error
		dstTx, err = dstChain.SendIBCTransfer(ctx, channelId, dstUser.KeyName, testCoinDstToSrc, nil)
		if err != nil {
			return fmt.Errorf("failed to send ibc transfer from destination: %w", err)
		}
		return nil
	})

	req.NoError(eg.Wait())
	req.NoError(srcTx.Validate(), "source ibc transfer tx is invalid")
	req.NoError(dstTx.Validate(), "destination ibc transfer tx is invalid")

	channels, err := relayer.GetChannels(ctx, eRep, srcChain.Config().ChainID)

	req.NoError(err, fmt.Sprintf("failed to get channels: %s", err))
	req.Len(channels, 1, fmt.Sprintf("channel count invalid. expected: 1, actual: %d", len(channels)))

	err = relayer.StartRelayer(ctx, eRep, testconfig.TestPath)
	req.NoError(err, fmt.Sprintf("failed to start relayer: %s", err))

	// wait for relayer to start up
	time.Sleep(5 * time.Second)

	t.Cleanup(func() {
		if err := relayer.StopRelayer(ctx, eRep); err != nil {
			t.Logf("error stopping relayer: %v", err)
		}
	})

	srcAck, err := test.PollForAck(ctx, srcChain, srcTx.Height, srcTx.Height+pollHeightMax, srcTx.Packet)
	req.NoError(err, "failed to get acknowledgement on source chain")
	req.NoError(srcAck.Validate(), "invalid acknowledgement on source chain")

	dstAck, err := test.PollForAck(ctx, dstChain, dstTx.Height, dstTx.Height+pollHeightMax, dstTx.Packet)
	req.NoError(err, "failed to get acknowledgement on destination chain")
	req.NoError(dstAck.Validate(), "invalid acknowledgement on source chain")

	// get ibc denom for dst denom on src chain
	srcDemonTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", "channel-0", srcChainCfg.Denom))
	dstIbcDenom := srcDemonTrace.IBCDenom()

	srcFinalBalance, err := srcChain.GetBalance(ctx, srcUser.Bech32Address(srcChainCfg.Bech32Prefix), srcChainCfg.Denom)
	req.NoError(err, "failed to get balance from source chain")

	dstFinalBalance, err := dstChain.GetBalance(ctx, srcUser.Bech32Address(dstChainCfg.Bech32Prefix), dstIbcDenom)
	req.NoError(err, "failed to get balance from dest chain")

	totalFees := srcChain.GetGasFeesInNativeDenom(srcTx.GasSpent)
	expectedDifference := testCoinSrcToDst.Amount + totalFees

	srcInitialBalance := int64(10_000_000)
	req.Equal(srcInitialBalance-expectedDifference, srcFinalBalance, "source address should have paid the full amount + gas fees")
	req.Equal(testCoinSrcToDst.Amount, dstFinalBalance, "destination address should be match the amount sent")
}
