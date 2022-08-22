package testconfig

import (
	"fmt"
	"os"

	"github.com/strangelove-ventures/ibctest/ibc"
)

const (
	// ChainImageEnv specifies the image that the chains will use. If left unspecified, it will
	// default to being determined based on the specified binary. E.g. ghcr.io/cosmos/ibc-go-simd
	ChainImageEnv = "CHAIN_IMAGE"
	// ChainATagEnv specifies the tag that Chain A will use.
	ChainATagEnv = "CHAIN_A_TAG"
	// ChainBTagEnv specifies the tag that Chain B will use. If unspecified
	// the value will default to the same value as Chain A.
	ChainBTagEnv = "CHAIN_B_TAG"
	// GoRelayerTagEnv specifies the go relayer version. Defaults to "main"
	GoRelayerTagEnv = "RLY_TAG"
	// ChainBinaryEnv binary is the binary that will be used for both chains.
	ChainBinaryEnv = "CHAIN_BINARY"
	// defaultBinary is the default binary that will be used by the chains.
	defaultBinary = "simd"
	// defaultRlyTag is the tag that will be used if no relayer tag is specified.
	defaultRlyTag = "main"
)

func getChainImage(binary string) string {
	if binary == "" {
		binary = defaultBinary
	}
	return fmt.Sprintf("ghcr.io/cosmos/ibc-go-%s", binary)
}

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
	chainBinary, ok := os.LookupEnv(ChainBinaryEnv)
	if !ok {
		chainBinary = defaultBinary
	}

	chainATag, ok := os.LookupEnv(ChainATagEnv)
	if !ok {
		panic(fmt.Sprintf("must specify %s version for test with environment variable [%s]", chainBinary, ChainATagEnv))
	}

	chainBTag, ok := os.LookupEnv(ChainBTagEnv)
	if !ok {
		chainBTag = chainATag
	}

	rlyTag, ok := os.LookupEnv(GoRelayerTagEnv)
	if !ok {
		rlyTag = defaultRlyTag
	}

	chainAImage := getChainImage(chainBinary)
	specifiedChainImage, ok := os.LookupEnv(ChainImageEnv)
	if ok {
		chainAImage = specifiedChainImage
	}
	chainBImage := chainAImage

	return TestConfig{
		ChainAConfig: ChainConfig{
			Image:  chainAImage,
			Tag:    chainATag,
			Binary: chainBinary,
		},
		ChainBConfig: ChainConfig{
			Image:  chainBImage,
			Tag:    chainBTag,
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
