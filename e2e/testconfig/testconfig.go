package testconfig

import (
	"fmt"
	"os"

	"github.com/strangelove-ventures/ibctest/ibc"
)

const (
	ChainASimdImageEnv = "CHAIN_A_SIMD_IMAGE"
	ChainASimdTag      = "CHAIN_A_SIMD_TAG"
	ChainBSimdImageEnv = "CHAIN_B_SIMD_IMAGE"
	ChainBSimdTagEnv   = "CHAIN_B_SIMD_TAG"
	GoRelayerTagEnv    = "RLY_TAG"
	ChainABinaryEnv    = "CHAIN_A_BINARY"
	ChainBBinaryEnv    = "CHAIN_B_BINARY"

	defaultBinary    = "simd"
	defaultSimdImage = "ghcr.io/cosmos/ibc-go-simd-e2e"
	defaultRlyTag    = "main"
)

// TestConfig holds various fields used in the E2E tests.
type TestConfig struct {
	ChainAConfig ChainConfig
	ChainBConfig ChainConfig
	RlyTag       string
}

type ChainConfig struct {
	Image  string
	Tag    string
	Binary string
}

// FromEnv returns a TestConfig constructed from environment variables.
func FromEnv() TestConfig {
	chainABinary, ok := os.LookupEnv(ChainABinaryEnv)
	if !ok {
		chainABinary = defaultBinary
	}

	chainBBinary, ok := os.LookupEnv(ChainBBinaryEnv)
	if !ok {
		chainBBinary = chainABinary
	}

	chainASimdImage, ok := os.LookupEnv(ChainASimdImageEnv)
	if !ok {
		chainASimdImage = defaultSimdImage
	}

	chainASimdTag, ok := os.LookupEnv(ChainASimdTag)
	if !ok {
		panic(fmt.Sprintf("must specify %s version for test with environment variable [%s]", chainABinary, ChainASimdTag))
	}

	chainBSimdImage, ok := os.LookupEnv(ChainBSimdImageEnv)
	if !ok {
		chainBSimdImage = chainASimdImage
	}

	chainBSimdTag, ok := os.LookupEnv(ChainBSimdTagEnv)
	if !ok {
		chainBSimdTag = chainASimdTag
	}

	rlyTag, ok := os.LookupEnv(GoRelayerTagEnv)
	if !ok {
		rlyTag = defaultRlyTag
	}

	return TestConfig{
		ChainAConfig: ChainConfig{
			Image:  chainASimdImage,
			Tag:    chainASimdTag,
			Binary: chainABinary,
		},
		ChainBConfig: ChainConfig{
			Image:  chainBSimdImage,
			Tag:    chainBSimdTag,
			Binary: chainBBinary,
		},
		RlyTag: rlyTag,
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
	chainACfg := newDefaultSimappConfig(tc.ChainAConfig, "simapp-a", "chain-a", "atoma")
	chainBCfg := newDefaultSimappConfig(tc.ChainBConfig, "simapp-b", "chain-b", "atomb")
	return ChainOptions{
		ChainAConfig: &chainACfg,
		ChainBConfig: &chainBCfg,
	}
}

// newDefaultSimappConfig creates an ibc configuration for simd.
func newDefaultSimappConfig(cc ChainConfig, name, chainID, denom string) ibc.ChainConfig {
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    name,
		ChainID: chainID,
		Images: []ibc.DockerImage{
			{
				Repository: cc.Image,
				Version:    cc.Tag,
			},
		},
		Bin:            cc.Binary,
		Bech32Prefix:   "cosmos",
		Denom:          denom,
		GasPrices:      fmt.Sprintf("0.00%s", denom),
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
	}
}
