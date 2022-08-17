package testconfig

import (
	"fmt"
	"os"

	"github.com/strangelove-ventures/ibctest/ibc"
)

const (
	// ChainASimdImageEnv specifies the image that Chain A will use.
	ChainASimdImageEnv = "CHAIN_A_SIMD_IMAGE"
	// ChainASimdTagEnv specifies the tag that Chain A will use.
	ChainASimdTagEnv = "CHAIN_A_SIMD_TAG"
	// ChainBSimdImageEnv specifies the image that Chain B will use. If unspecified
	// the value will default to the same value as Chain A.
	ChainBSimdImageEnv = "CHAIN_B_SIMD_IMAGE"
	// ChainBSimdTagEnv specifies the tag that Chain B will use. If unspecified
	// the value will default to the same value as Chain A.
	ChainBSimdTagEnv = "CHAIN_B_SIMD_TAG"
	// GoRelayerTagEnv specifies the go relayer version. Defaults to "main"
	GoRelayerTagEnv = "RLY_TAG"
	// ChainBinary binary is the binary that will be used for the chains.
	ChainBinary = "CHAIN_BINARY"
	// defaultBinary is the default binary that will be used by the chains.
	defaultBinary = "simd"
	// defaultSimdImage is the default image that will be used for the chain if none are specified.
	defaultSimdImage = "ghcr.io/cosmos/ibc-go-simd"
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
	chainBinary, ok := os.LookupEnv(ChainBinary)
	if !ok {
		chainBinary = defaultBinary
	}

	chainASimdImage, ok := os.LookupEnv(ChainASimdImageEnv)
	if !ok {
		chainASimdImage = defaultSimdImage
	}

	chainASimdTag, ok := os.LookupEnv(ChainASimdTagEnv)
	if !ok {
		panic(fmt.Sprintf("must specify %s version for test with environment variable [%s]", chainBinary, ChainASimdTagEnv))
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
			Binary: chainBinary,
		},
		ChainBConfig: ChainConfig{
			Image:  chainBSimdImage,
			Tag:    chainBSimdTag,
			Binary: chainBinary,
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

// SetChainBinaryVersions is a helper function for local cross-version testing
func SetChainBinaryVersions(chainaSimdImg, chainaSimdTag, chainBinary, chainbSimdImg, chainbSimdTag string) {
	os.Setenv("CHAIN_A_SIMD_IMAGE", chainaSimdImg)
	os.Setenv("CHAIN_A_SIMD_TAG", chainaSimdTag)
	os.Setenv("CHAIN_B_SIMD_IMAGE", chainbSimdImg)
	os.Setenv("CHAIN_B_SIMD_TAG", chainbSimdTag)
	os.Setenv("CHAIN_BINARY", chainBinary)
}
