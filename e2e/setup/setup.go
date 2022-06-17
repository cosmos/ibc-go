package setup

import (
	"context"
	"fmt"
	"github.com/cosmos/ibc-go/v3/e2e/testconfig"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"testing"
)

func newSimappConfig(name, chainId, denom string) ibc.ChainConfig {
	tc := testconfig.FromEnv()
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    name,
		ChainID: chainId,
		Images: []ibc.DockerImage{
			{
				Repository: tc.SimdImage,
				Version:    tc.SimdTag,
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

// StandardTwoChainEnvironment creates two default simapp containers as well as a go relayer container.
// the relayer that is returned is not yet started.
func StandardTwoChainEnvironment(t *testing.T, req *require.Assertions, eRep *testreporter.RelayerExecReporter) (*cosmos.CosmosChain, *cosmos.CosmosChain, ibc.Relayer) {
	ctx := context.Background()
	pool, network := ibctest.DockerSetup(t)
	home := t.TempDir() // Must be before chain cleanup to avoid test error during cleanup.

	l := zap.NewExample()
	srcChain := cosmos.NewCosmosChain(t.Name(), newSimappConfig("simapp-a", "chain-a", "atoma"), 1, 1, l)
	dstChain := cosmos.NewCosmosChain(t.Name(), newSimappConfig("simapp-b", "chain-b", "atomb"), 1, 1, l)

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

	ic := ibctest.NewInterchain().
		AddChain(srcChain).
		AddChain(dstChain).
		AddRelayer(r, "r").
		AddLink(ibctest.InterchainLink{
			Chain1:  srcChain,
			Chain2:  dstChain,
			Relayer: r,
			Path:    testconfig.TestPath,
		})

	req.NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:  t.Name(),
		HomeDir:   home,
		Pool:      pool,
		NetworkID: network,
	}))

	return srcChain, dstChain, r
}
