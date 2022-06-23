package setup

import (
	"fmt"
	"github.com/cosmos/ibc-go/v3/e2e/testconfig"
	"github.com/ory/dockertest/v3"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/relayer"
	"go.uber.org/zap"
	"testing"
)

func NewRelayer(t *testing.T, logger *zap.Logger, pool *dockertest.Pool, network string, home string) ibc.Relayer {
	r := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main")).Build(
		t, pool, network, home,
	)
	return r
}

// NewSimappConfig creates an ibc configuration for simd.
func NewSimappConfig(tc testconfig.TestConfig, name, chainId, denom string) ibc.ChainConfig {
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
