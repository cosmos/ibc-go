package e2e

import (
	"context"
	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"testing"
)

var ibcChainAConfig ibc.ChainConfig
var ibcChainBConfig ibc.ChainConfig

func init() {
	ibcChainAConfig = ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "simmapp-a",
		ChainID: "chain-a",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/cosmos/ibc-go-simd",
				Version:    "v3.0.0",
			},
		},
		Bin:            "simd",
		Bech32Prefix:   "cosmos",
		Denom:          "atoma",
		GasPrices:      "0.01atoma",
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
	}
	ibcChainBConfig = ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "simmapp-b",
		ChainID: "chain-b",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/cosmos/ibc-go-simd",
				Version:    "v3.0.0",
			},
		},
		Bin:            "simd",
		Bech32Prefix:   "cosmos",
		Denom:          "atomb",
		GasPrices:      "0.01atomb",
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
	}
}

const (
	pollHeightMax = uint64(50)
)

func TestIBCTest(t *testing.T) {

	pool, network := ibctest.DockerSetup(t)
	home := t.TempDir() // Must be before chain cleanup to avoid test error during cleanup.

	l := zap.NewExample()
	srcChain := cosmos.NewCosmosChain(t.Name(), ibcChainAConfig, 1, 1, l)
	dstChain := cosmos.NewCosmosChain(t.Name(), ibcChainBConfig, 1, 1, l)

	srcChainCfg := srcChain.Config()
	dstChainCfg := dstChain.Config()

	ctx := context.Background()
	t.Cleanup(func() {
		if err := srcChain.Cleanup(ctx); err != nil {
			t.Logf("Chain cleanup for %s failed: %v", srcChain.Config().ChainID, err)
		}
		if err := dstChain.Cleanup(ctx); err != nil {
			t.Logf("Chain cleanup for %s failed: %v", dstChain.Config().ChainID, err)
		}
	})

	r := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(
		t, pool, network, home,
	)

	pathName := "p"
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

	err := test.WaitForBlocks(ctx, 10, srcChain)
	req.NoError(err, "simapp chain a failed to make blocks")
	err = test.WaitForBlocks(ctx, 10, dstChain)
	req.NoError(err, "simapp chain b failed to make blocks")

	users := ibctest.GetAndFundTestUsers(t, ctx, "prefix", 10000, srcChain, dstChain)

	err = r.CreateChannel(ctx, eRep, pathName, ibc.CreateChannelOptions{
		SourcePortName: "transfer",
		DestPortName:   "transfer",
		Order:          "unordered",
		Version:        "ics20-1",
	})
	req.NoError(err)

	srcUser := users[0]
	dstUser := users[1]

	//userAddressBytes, err := dstChain.GetAddress(ctx, dstUser.KeyName)
	//req.NoError(err)

	//toUserAddress, err := types.Bech32ifyAddressBytes(dstChain.Config().Bech32Prefix, userAddressBytes)
	//
	//dstDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", "channel-0", chaina.Config().Denom))
	//bal, err := chainb.GetBalance(ctx, toUserAddress, dstDenomTrace.IBCDenom())

	//t.Logf("BEFORE: bal: %d", bal)
	//
	tx, err := srcChain.SendIBCTransfer(ctx, "channel-0", srcUser.KeyName, ibc.WalletAmount{
		Address: srcUser.Bech32Address(dstChainCfg.Bech32Prefix),
		Denom:   srcChain.Config().Denom,
		Amount:  1000,
	}, nil)

	req.NoError(err)
	req.NoError(tx.Validate())

	err = test.WaitForBlocks(ctx, 10, srcChain)
	req.NoError(err, "simapp chain a failed to make blocks")
	err = test.WaitForBlocks(ctx, 10, dstChain)
	req.NoError(err, "simapp chain b failed to make blocks")

	// get ibc denom for dst denom on src chain
	srcDemonTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", "channel-0", srcChainCfg.Denom))
	dstIbcDenom := srcDemonTrace.IBCDenom()

	srcFinalBalance, err := srcChain.GetBalance(ctx, dstUser.Bech32Address(srcChainCfg.Bech32Prefix), srcChainCfg.Denom)
	req.NoError(err, "failed to get balance from source chain")

	dstFinalBalance, err := dstChain.GetBalance(ctx, dstUser.Bech32Address(dstChainCfg.Bech32Prefix), dstIbcDenom)
	req.NoError(err, "failed to get balance from dest chain")

	//bal, err = chainb.GetBalance(ctx, toUserAddress, dstDenomTrace.IBCDenom())
	t.Logf("SRC: %d", srcFinalBalance)
	t.Logf("DST: %d", dstFinalBalance)

	req.NoError(err)
	//t.Logf("AFTER: bal: %d", bal)

	req.NotEmpty(dstFinalBalance)

}
