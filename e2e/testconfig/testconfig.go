package testconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tmjson "github.com/cometbft/cometbft/libs/json"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"

	"github.com/cosmos/ibc-go/e2e/relayer"
	"github.com/cosmos/ibc-go/e2e/semverutil"
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
	// RelayerTagEnv specifies the relayer version. Defaults to "main"
	RelayerTagEnv = "RELAYER_TAG"
	// RelayerTypeEnv specifies the type of relayer that should be used.
	RelayerTypeEnv = "RELAYER_TYPE"
	// ChainBinaryEnv binary is the binary that will be used for both chains.
	ChainBinaryEnv = "CHAIN_BINARY"
	// ChainUpgradeTagEnv specifies the upgrade version tag
	ChainUpgradeTagEnv = "CHAIN_UPGRADE_TAG"
	// ChainUpgradePlanEnv specifies the upgrade plan name
	ChainUpgradePlanEnv = "CHAIN_UPGRADE_PLAN"

	// defaultBinary is the default binary that will be used by the chains.
	defaultBinary = "simd"
	// defaultRlyTag is the tag that will be used if no relayer tag is specified.
	// all images are here https://github.com/cosmos/relayer/pkgs/container/relayer/versions
	defaultRlyTag = "andrew-tendermint_v0.37" // "v2.2.0"
	// defaultChainTag is the tag that will be used for the chains if none is specified.
	defaultChainTag = "main"
	// defaultRelayerType is the default relayer that will be used if none is specified.
	defaultRelayerType = relayer.Rly
)

func getChainImage(binary string) string {
	if binary == "" {
		binary = defaultBinary
	}
	return fmt.Sprintf("ghcr.io/cosmos/ibc-go-%s", binary)
}

// TestConfig holds various fields used in the E2E tests.
type TestConfig struct {
	ChainAConfig    ChainConfig
	ChainBConfig    ChainConfig
	RelayerConfig   relayer.Config
	UpgradeTag      string
	UpgradePlanName string
}

// ChainConfig holds information about an individual chain used in the tests.
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

	upgradePlan, ok := os.LookupEnv(ChainUpgradePlanEnv)
	if !ok {
		upgradePlan = ""
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
		UpgradeTag:      upgradeTag,
		UpgradePlanName: upgradePlan,
		RelayerConfig:   GetRelayerConfigFromEnv(),
	}
}

// GetRelayerConfigFromEnv returns the RelayerConfig from present environment variables.
func GetRelayerConfigFromEnv() relayer.Config {
	relayerType := strings.TrimSpace(os.Getenv(RelayerTypeEnv))
	if relayerType == "" {
		relayerType = defaultRelayerType
	}

	rlyTag := strings.TrimSpace(os.Getenv(RelayerTagEnv))
	if rlyTag == "" {
		if relayerType == relayer.Rly {
			rlyTag = defaultRlyTag
		}
		if relayerType == relayer.Hermes {
			// TODO: set default hermes version
		}
	}
	return relayer.Config{
		Tag:  rlyTag,
		Type: relayerType,
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

// IsCI returns true if the tests are running in CI, false is returned
// if the tests are running locally.
// Note: github actions passes a CI env value of true by default to all runners.
func IsCI() bool {
	return strings.ToLower(os.Getenv("CI")) == "true"
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
		CoinType:       fmt.Sprint(sdk.GetConfig().GetCoinType()),
		Denom:          denom,
		GasPrices:      fmt.Sprintf("0.00%s", denom),
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
		ModifyGenesis:  getGenesisModificationFunction(cc),
	}
}

// getGenesisModificationFunction returns a genesis modification function that handles the GenesisState type
// correctly depending on if the govv1beta1 gov module is used or if govv1 is being used.
func getGenesisModificationFunction(cc ChainConfig) func(ibc.ChainConfig, []byte) ([]byte, error) {
	version := cc.Tag

	if govGenesisFeatureReleases.IsSupported(version) {
		return defaultGovv1ModifyGenesis()
	}

	return defaultGovv1Beta1ModifyGenesis()
}

// govGenesisFeatureReleases represents the releases the governance module genesis
// was upgraded from v1beta1 to v1.
var govGenesisFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v7",
}

// defaultGovv1ModifyGenesis will only modify governance params to ensure the voting period and minimum deposit
// are functional for e2e testing purposes.
func defaultGovv1ModifyGenesis() func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		genDoc, err := tmtypes.GenesisDocFromJSON(genbz)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into genesis doc: %w", err)
		}

		var appState genutiltypes.AppMap
		if err := json.Unmarshal(genDoc.AppState, &appState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into app state: %w", err)
		}

		govGenBz, err := modifyGovAppState(chainConfig, appState[govtypes.ModuleName])
		if err != nil {
			return nil, err
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

// defaultGovv1Beta1ModifyGenesis will only modify governance params to ensure the voting period and minimum deposit
// // are functional for e2e testing purposes.
func defaultGovv1Beta1ModifyGenesis() func(ibc.ChainConfig, []byte) ([]byte, error) {
	const appStateKey = "app_state"
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		genesisDocMap := map[string]interface{}{}
		err := json.Unmarshal(genbz, &genesisDocMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into genesis doc: %w", err)
		}

		appStateMap, ok := genesisDocMap[appStateKey].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("failed to extract to app_state")
		}

		govModuleBytes, err := json.Marshal(appStateMap[govtypes.ModuleName])
		if err != nil {
			return nil, fmt.Errorf("failed to extract gov genesis bytes: %s", err)
		}

		govModuleGenesisBytes, err := modifyGovv1Beta1AppState(chainConfig, govModuleBytes)
		if err != nil {
			return nil, err
		}

		govModuleGenesisMap := map[string]interface{}{}
		err = json.Unmarshal(govModuleGenesisBytes, &govModuleGenesisMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal gov genesis bytes into map: %w", err)
		}

		appStateMap[govtypes.ModuleName] = govModuleGenesisMap
		genesisDocMap[appStateKey] = appStateMap

		finalGenesisDocBytes, err := json.MarshalIndent(genesisDocMap, "", " ")
		if err != nil {
			return nil, err
		}

		return finalGenesisDocBytes, nil
	}
}

// modifyGovAppState takes the existing gov app state and marshals it to a govv1 GenesisState.
func modifyGovAppState(chainConfig ibc.ChainConfig, govAppState []byte) ([]byte, error) {
	cfg := testutil.MakeTestEncodingConfig()

	cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)
	govv1.RegisterInterfaces(cfg.InterfaceRegistry)

	govGenesisState := &govv1.GenesisState{}

	if err := cdc.UnmarshalJSON(govAppState, govGenesisState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis bytes into gov genesis state: %w", err)
	}

	if govGenesisState.Params == nil {
		govGenesisState.Params = &govv1.Params{}
	}

	govGenesisState.Params.MinDeposit = sdk.NewCoins(sdk.NewCoin(chainConfig.Denom, govv1beta1.DefaultMinDepositTokens))
	vp := testvalues.VotingPeriod
	govGenesisState.Params.VotingPeriod = &vp

	govGenBz, err := cdc.MarshalJSON(govGenesisState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gov genesis state: %w", err)
	}

	return govGenBz, nil
}

// modifyGovv1Beta1AppState takes the existing gov app state and marshals it to a govv1beta1 GenesisState.
func modifyGovv1Beta1AppState(chainConfig ibc.ChainConfig, govAppState []byte) ([]byte, error) {
	cfg := testutil.MakeTestEncodingConfig()

	cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)
	govv1beta1.RegisterInterfaces(cfg.InterfaceRegistry)

	govGenesisState := &govv1beta1.GenesisState{}
	if err := cdc.UnmarshalJSON(govAppState, govGenesisState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis bytes into govv1beta1 genesis state: %w", err)
	}

	govGenesisState.DepositParams.MinDeposit = sdk.NewCoins(sdk.NewCoin(chainConfig.Denom, govv1beta1.DefaultMinDepositTokens))
	govGenesisState.VotingParams.VotingPeriod = testvalues.VotingPeriod

	govGenBz, err := cdc.MarshalJSON(govGenesisState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gov genesis state: %w", err)
	}

	return govGenBz, nil
}
