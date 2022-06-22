package setup

import (
	"context"
	"fmt"
	"github.com/cosmos/ibc-go/v3/e2e/testconfig"
	"github.com/ory/dockertest/v3"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/relayer"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"testing"
)

func NewRelayer(t *testing.T, logger *zap.Logger, pool *dockertest.Pool, network string, home string) ibc.Relayer {
	r := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main")).Build(
		t, pool, network, home,
	)
	return r
}

// StandardTwoChainEnvironment creates two default simapp containers as well as a go relayer container.
// the relayer that is returned is not yet started.
func StandardTwoChainEnvironment(t *testing.T, req *require.Assertions, eRep *testreporter.RelayerExecReporter, optFuncs ...ConfigurationFunc) (*cosmos.CosmosChain, *cosmos.CosmosChain, ibc.Relayer) {
	opts := defaultSetupOpts()
	for _, fn := range optFuncs {
		fn(opts)
	}

	ctx := context.Background()
	pool, network := ibctest.DockerSetup(t)
	home := t.TempDir() // Must be before chain cleanup to avoid test error during cleanup.

	logger := zaptest.NewLogger(t)
	chain1 := cosmos.NewCosmosChain(t.Name(), *opts.ChainAConfig, 1, 0, logger)
	chain2 := cosmos.NewCosmosChain(t.Name(), *opts.ChainBConfig, 1, 0, logger)

	t.Cleanup(func() {
		if !t.Failed() {
			for _, c := range []*cosmos.CosmosChain{chain1, chain2} {
				if err := c.Cleanup(ctx); err != nil {
					t.Logf("Chain cleanup for %s failed: %v", c.Config().ChainID, err)
				}
			}
		}
	})

	r := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main")).Build(
		t, pool, network, home,
	)

	ic := ibctest.NewInterchain().
		AddChain(chain1).
		AddChain(chain2).
		AddRelayer(r, "r").
		AddLink(ibctest.InterchainLink{
			Chain1:  chain1,
			Chain2:  chain2,
			Relayer: r,
			Path:    testconfig.TestPath,
		})

	req.NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:         t.Name(),
		HomeDir:          home,
		Pool:             pool,
		NetworkID:        network,
		SkipPathCreation: true,
	}))

	// all channels & connections were created in ic.Build
	//if !opts.SkipPathCreation {
	return chain1, chain2, r
	//}
	//
	//req.NoError(r.GeneratePath(ctx, eRep, chain1.Config().ChainID, chain2.Config().ChainID, testconfig.TestPath))
	//req.NoError(r.CreateClients(ctx, eRep, testconfig.TestPath))
	//
	//// The client isn't created immediately -- wait for two blocks to ensure the clients are ready.
	//req.NoError(test.WaitForBlocks(ctx, 2, chain1, chain2))
	//req.NoError(r.CreateConnections(ctx, eRep, testconfig.TestPath))
	//req.NoError(r.CreateChannel(ctx, eRep, testconfig.TestPath, *opts.CreateChannelOptions))

	//return chain1, chain2, r
}

// ConfigurationFunc allows for arbitrary configuration of the setup Options.
type ConfigurationFunc func(opts *Options)

// Options holds values that allow for configuring setup functions.
type Options struct {
	ChainAConfig         *ibc.ChainConfig
	ChainBConfig         *ibc.ChainConfig
	SkipPathCreation     bool
	CreateChannelOptions *ibc.CreateChannelOptions
}

func defaultSetupOpts() *Options {
	chainAConfig := NewSimappConfig("simapp-a", "chain-a", "atoma")
	chainBConfig := NewSimappConfig("simapp-b", "chain-b", "atomb")
	return &Options{
		ChainAConfig:     &chainAConfig,
		ChainBConfig:     &chainBConfig,
		SkipPathCreation: false,
		CreateChannelOptions: &ibc.CreateChannelOptions{
			SourcePortName: "transfer",
			DestPortName:   "transfer",
			Order:          "unordered",
			Version:        "ics20-1",
		},
	}
}

// NewSimappConfig creates an ibc configuration for simd.
func NewSimappConfig(name, chainId, denom string) ibc.ChainConfig {
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

// FeeMiddlewareOptions configures both of the chains to have fee middleware enabled.
func FeeMiddlewareOptions() func(*Options) {
	return func(opts *Options) {
		opts.CreateChannelOptions.Version = "{\"fee_version\":\"ics29-1\",\"app_version\":\"ics20-1\"}"
		opts.CreateChannelOptions.DestPortName = "transfer"
		opts.CreateChannelOptions.SourcePortName = "transfer"
		opts.SkipPathCreation = true
	}
}
