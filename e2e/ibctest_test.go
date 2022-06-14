package e2e

import (
	"context"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
)

var ibcChainConfig ibc.ChainConfig

func init() {
	ibcChainConfig = ibc.ChainConfig{
		Type:    "cosmos",
		Name:    "simmapp",
		ChainID: "chain-a",
		Images: []ibc.DockerImage{
			{
				Repository: "ghcr.io/cosmos/ibc-go-simd",
				Version:    "v3.0.0",
			},
		},
		Bin:            "simd",
		Bech32Prefix:   "cosmos",
		Denom:          "atom",
		GasPrices:      "0.01atom",
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
	}
}

func TestIBCTest(t *testing.T) {

	pool, network := ibctest.DockerSetup(t)
	home := t.TempDir() // Must be before chain cleanup to avoid test error during cleanup.

	l := zap.NewExample()
	chain := cosmos.NewCosmosChain(t.Name(), ibcChainConfig, 4, 1, l)

	ctx := context.Background()
	t.Cleanup(func() {
		if err := chain.Cleanup(ctx); err != nil {
			t.Logf("Chain cleanup for %s failed: %v", chain.Config().ChainID, err)
		}
	})

	err := chain.Initialize(t.Name(), home, pool, network)
	require.NoError(t, err, "failed to initialize simapp chain")

	err = chain.Start(t.Name(), ctx)
	require.NoError(t, err, "failed to start simapp chain")

	err = test.WaitForBlocks(ctx, 10, chain)

	require.NoError(t, err, "simapp chain failed to make blocks")

}
