package testconfig

import (
	"fmt"
	"os"

	"github.com/strangelove-ventures/ibctest/ibc"
)

const (
	DefaultSimdImage = "ghcr.io/cosmos/ibc-go-simd-e2e"
	SimdImageEnv     = "SIMD_IMAGE"
	SimdTagEnv       = "SIMD_TAG"
	GoRelayerTag     = "RLY_TAG"

	defaultRlyTag = "main"
)

// TestConfig holds various fields used in the E2E tests.
type TestConfig struct {
	SimdImage string
	SimdTag   string
	RlyTag    string
}

// FromEnv returns a TestConfig constructed from environment variables.
func FromEnv() TestConfig {
	simdImage, ok := os.LookupEnv(SimdImageEnv)
	if !ok {
		simdImage = DefaultSimdImage
	}

	simdTag, ok := os.LookupEnv(SimdTagEnv)
	if !ok {
		panic(fmt.Sprintf("must specify simd version for test with environment variable [%s]", SimdTagEnv))
	}

	rlyTag, ok := os.LookupEnv(GoRelayerTag)
	if !ok {
		rlyTag = defaultRlyTag
	}

	return TestConfig{
		SimdImage: simdImage,
		SimdTag:   simdTag,
		RlyTag:    rlyTag,
	}
}

// ChainOptions stores chain configurations for the chains that will be
// created for the tests. They can be modified by passing ChainOptionConfiguration
// to E2ETestSuite.GetChains.
type ChainOptions struct {
	ChainAConfig *ibc.ChainConfig
	ChainBConfig *ibc.ChainConfig
}

// ChainOptionConfiguration enables arbitrary configuration of ChainOptions.
type ChainOptionConfiguration func(options *ChainOptions)

// DefaultChainOptions returns the default configuration for the chains.
// These options can be configured by passing configuration functions to E2ETestSuite.GetChains.
func DefaultChainOptions() ChainOptions {
	tc := FromEnv()
	chainACfg := newDefaultSimappConfig(tc, "simapp-a", "chain-a", "atoma")
	chainBCfg := newDefaultSimappConfig(tc, "simapp-b", "chain-b", "atomb")
	return ChainOptions{
		ChainAConfig: &chainACfg,
		ChainBConfig: &chainBCfg,
	}
}

// newDefaultSimappConfig creates an ibc configuration for simd.
func newDefaultSimappConfig(tc TestConfig, name, chainID, denom string) ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    name,
		ChainID: chainID,
		Images: []ibc.DockerImage{
			{
				Repository: tc.SimdImage,
				Version:    tc.SimdTag,
			},
		},
		Bin:            "simd",
		Bech32Prefix:   "cosmos",
		Denom:          denom,
		GasPrices:      fmt.Sprintf("0.00%s", denom),
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
	}
}
