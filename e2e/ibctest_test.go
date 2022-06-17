package e2e

import (
	"context"
	"fmt"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/sync/errgroup"
	"testing"
	"time"
)

var ibcChainAConfig ibc.ChainConfig
var ibcChainBConfig ibc.ChainConfig

func newSimappConfig(name, chainId, denom string) ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    name,
		ChainID: chainId,
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/cosmos/ibc-go-simd",
				Version:    "v3.0.0",
			},
		},
		Bin:            "simd",
		Bech32Prefix:   "cosmos",
		Denom:          denom,
		GasPrices:      fmt.Sprintf("0.01%s", denom),
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
	}
}

func init() {
	ibcChainAConfig = newSimappConfig("simapp-a", "chain-a", "atoma")
	ibcChainBConfig = newSimappConfig("simapp-b", "chain-b", "atomb")
}

const (
	pollHeightMax = uint64(50)
)

func TestSimappIBCTest(t *testing.T) {

	pool, network := ibctest.DockerSetup(t)
	home := t.TempDir() // Must be before chain cleanup to avoid test error during cleanup.

	l := zap.NewExample()
	srcChain := cosmos.NewCosmosChain(t.Name(), ibcChainAConfig, 1, 1, l)
	dstChain := cosmos.NewCosmosChain(t.Name(), ibcChainBConfig, 1, 1, l)

	srcChainCfg := srcChain.Config()
	dstChainCfg := dstChain.Config()

	ctx := context.Background()
	t.Cleanup(func() {
		for _, c := range []*cosmos.CosmosChain{srcChain, dstChain} {
			if err := c.Cleanup(ctx); err != nil {
				t.Logf("Chain cleanup for %s failed: %v", c.Config().ChainID, err)
			}
		}
	})

	r := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(
		t, pool, network, home,
	)

	pathName := "test-path"
	ic := ibctest.NewInterchain().
		AddChain(srcChain).
		AddChain(dstChain).
		AddRelayer(r, "r").
		AddLink(ibctest.InterchainLink{
			Chain1:  srcChain,
			Chain2:  dstChain,
			Relayer: r,
			Path:    pathName,
		})

	rep := testreporter.NewNopReporter()
	req := require.New(rep.TestifyT(t))
	eRep := rep.RelayerExecReporter(t)

	req.NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:  t.Name(),
		HomeDir:   home,
		Pool:      pool,
		NetworkID: network,
	}))

	users := ibctest.GetAndFundTestUsers(t, ctx, "some-prefix", 10_000_000, srcChain, dstChain)

	srcUser := users[0]
	dstUser := users[1]

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

	require.NoError(t, eg.Wait())
	require.NoError(t, srcTx.Validate(), "source ibc transfer tx is invalid")
	require.NoError(t, dstTx.Validate(), "destination ibc transfer tx is invalid")

	channels, err := r.GetChannels(ctx, eRep, srcChain.Config().ChainID)

	req.NoError(err, fmt.Sprintf("failed to get channels: %s", err))
	req.Len(channels, 1, fmt.Sprintf("channel count invalid. expected: 1, actual: %d", len(channels)))
	t.Logf("channels: %+v", channels)

	err = r.StartRelayer(ctx, eRep, pathName)
	req.NoError(err, fmt.Sprintf("failed to start relayer: %s", err))

	// wait for relayer to start up
	time.Sleep(5 * time.Second)

	t.Cleanup(func() {
		if err := r.StopRelayer(ctx, eRep); err != nil {
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

	t.Logf("SRC: %d", srcFinalBalance)
	t.Logf("DST: %d", dstFinalBalance)

	req.NoError(err)
	req.NotEmpty(dstFinalBalance)
}
