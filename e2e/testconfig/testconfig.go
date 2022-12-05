package testconfig

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/cosmos/ibc-go/e2e/testvalues"
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
	// ChainUpgradeTagEnv specifies the upgrade version tag
	ChainUpgradeTagEnv = "CHAIN_UPGRADE_TAG"
	// defaultBinary is the default binary that will be used by the chains.
	defaultBinary = "simd"
	// defaultRlyTag is the tag that will be used if no relayer tag is specified.
	defaultRlyTag = "main"
	// defaultChainTag is the tag that will be used for the chains if none is specified.
	defaultChainTag = "main"
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
	UpgradeTag   string
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
		chainATag = defaultChainTag
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

	upgradeTag, ok := os.LookupEnv(ChainUpgradeTagEnv)
	if !ok {
		upgradeTag = ""
	}

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
		RlyTag:     rlyTag,
		UpgradeTag: upgradeTag,
	}
}

func GetChainATag() string {
	chainATag, ok := os.LookupEnv(ChainATagEnv)
	if !ok {
		panic(fmt.Sprintf("no environment variable specified for %s", ChainATagEnv))
	}
	return chainATag
}

func GetChainBTag() string {
	chainBTag, ok := os.LookupEnv(ChainBTagEnv)
	if !ok {
		return GetChainATag()
	}
	return chainBTag
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
		ModifyGenesis:  defaultModifyGenesis(),
	}
}

// defaultModifyGenesis will only modify governance params to ensure the voting period and minimum deposit
// are functional for e2e testing purposes.
func defaultModifyGenesis() func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		genDoc, err := tmtypes.GenesisDocFromJSON(genbz)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into genesis doc: %w", err)
		}

		var appState genutiltypes.AppMap
		if err := json.Unmarshal(genDoc.AppState, &appState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into app state: %w", err)
		}

		cfg := simappparams.MakeTestEncodingConfig()
		govv1beta1.RegisterInterfaces(cfg.InterfaceRegistry)
		cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)

		govGenesisState := &govv1beta1.GenesisState{}
		if err := cdc.UnmarshalJSON(appState[govtypes.ModuleName], govGenesisState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into gov genesis state: %w", err)
		}

		// set correct minimum deposit using configured denom
		govGenesisState.DepositParams.MinDeposit = sdk.NewCoins(sdk.NewCoin(chainConfig.Denom, govv1beta1.DefaultMinDepositTokens))
		govGenesisState.VotingParams.VotingPeriod = testvalues.VotingPeriod

		govGenBz, err := cdc.MarshalJSON(govGenesisState)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal gov genesis state: %w", err)
		}

		appState[govtypes.ModuleName] = govGenBz

		genDoc.AppState, err = json.Marshal(appState)
		if err != nil {
			return nil, err
		}

		bz, err := tmjson.MarshalIndent(genDoc, "", "  ")
		if err != nil {
			return nil, err
		}

		return bz, nil
	}
}
