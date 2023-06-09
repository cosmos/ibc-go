package testconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
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
	interchaintestutil "github.com/strangelove-ventures/interchaintest/v7/testutil"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/ibc-go/e2e/relayer"
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
	// E2EConfigFilePathEnv allows you to specify a custom path for the config file to be used.
	E2EConfigFilePathEnv = "E2E_CONFIG_PATH"

	// defaultBinary is the default binary that will be used by the chains.
	defaultBinary = "simd"
	// defaultRlyTag is the tag that will be used if no relayer tag is specified.
	// all images are here https://github.com/cosmos/relayer/pkgs/container/relayer/versions
	defaultRlyTag = "latest" // "andrew-tendermint_v0.37" // "v2.2.0"
	// defaultHermesTag is the tag that will be used if no relayer tag is specified for hermes.
	defaultHermesTag = "v1.4.0"
	// defaultChainTag is the tag that will be used for the chains if none is specified.
	defaultChainTag = "main"
	// defaultRelayerType is the default relayer that will be used if none is specified.
	defaultRelayerType = relayer.Rly
	// defaultConfigFileName is the default filename for the config file that can be used to configure
	// e2e tests. See sample.config.yaml as an example for what this should look like.
	defaultConfigFileName = ".ibc-go-e2e-config.yaml"

	// icadBinary is the binary for interchain-accounts-demo repository.
	icadBinary = "icad"
)

func getChainImage(binary string) string {
	if binary == "" {
		binary = defaultBinary
	}
	return fmt.Sprintf("ghcr.io/cosmos/ibc-go-%s", binary)
}

// TestConfig holds configuration used throughout the different e2e tests.
type TestConfig struct {
	// ChainConfigs holds configuration values related to the chains used in the tests.
	ChainConfigs []ChainConfig `yaml:"chains"`
	// RelayerConfig holds configuration for the relayer to be used.
	RelayerConfig relayer.Config `yaml:"relayer"`
	// UpgradeConfig holds values used only for the upgrade tests.
	UpgradeConfig UpgradeConfig `yaml:"upgrade"`
	// CometBFTConfig holds values for configuring CometBFT.
	CometBFTConfig CometBFTConfig `yaml:"cometbft"`
	// DebugConfig holds configuration for miscellaneous options.
	DebugConfig DebugConfig `yaml:"debug"`
}

// GetChainNumValidators returns the number of validators for the specific chain index.
// default 1
func (tc TestConfig) GetChainNumValidators(idx int) int {
	if tc.ChainConfigs[idx].NumValidators > 0 {
		return tc.ChainConfigs[idx].NumValidators
	}
	return 1
}

// GetChainNumFullNodes returns the number of full nodes for the specific chain index.
// default 0
func (tc TestConfig) GetChainNumFullNodes(idx int) int {
	if tc.ChainConfigs[idx].NumFullNodes > 0 {
		return tc.ChainConfigs[idx].NumFullNodes
	}
	return 0
}

// GetChainAID returns the chain-id for chain A.
func (tc TestConfig) GetChainAID() string {
	if tc.ChainConfigs[0].ChainID != "" {
		return tc.ChainConfigs[0].ChainID
	}
	return "chain-a"
}

// GetChainBID returns the chain-id for chain B.
func (tc TestConfig) GetChainBID() string {
	if tc.ChainConfigs[1].ChainID != "" {
		return tc.ChainConfigs[1].ChainID
	}
	return "chain-b"
}

// UpgradeConfig holds values relevant to upgrade tests.
type UpgradeConfig struct {
	PlanName string `yaml:"planName"`
	Tag      string `yaml:"tag"`
}

// ChainConfig holds information about an individual chain used in the tests.
type ChainConfig struct {
	ChainID       string `yaml:"chainId"`
	Image         string `yaml:"image"`
	Tag           string `yaml:"tag"`
	Binary        string `yaml:"binary"`
	NumValidators int    `yaml:"numValidators"`
	NumFullNodes  int    `yaml:"numFullNodes"`
}

type CometBFTConfig struct {
	LogLevel string `yaml:"logLevel"`
}

type DebugConfig struct {
	// DumpLogs forces the logs to be collected before removing test containers.
	DumpLogs bool `yaml:"dumpLogs"`
}

// LoadConfig attempts to load a atest configuration from the default file path.
// if any environment variables are specified, they will take precedence over the individual configuration
// options.
func LoadConfig() TestConfig {
	fileTc, foundFile := fromFile()
	if !foundFile {
		return fromEnv()
	}
	return applyEnvironmentVariableOverrides(fileTc)
}

// fromFile returns a TestConfig from a json file and a boolean indicating if the file was found.
func fromFile() (TestConfig, bool) {
	var tc TestConfig
	bz, err := os.ReadFile(getConfigFilePath())
	if err != nil {
		return TestConfig{}, false
	}

	if err := yaml.Unmarshal(bz, &tc); err != nil {
		panic(err)
	}

	return tc, true
}

// applyEnvironmentVariableOverrides applies all environment variable changes to the config
// loaded from a file.
func applyEnvironmentVariableOverrides(fromFile TestConfig) TestConfig {
	envTc := fromEnv()

	if os.Getenv(ChainATagEnv) != "" {
		fromFile.ChainConfigs[0].Tag = envTc.ChainConfigs[0].Tag
	}

	if os.Getenv(ChainBTagEnv) != "" {
		fromFile.ChainConfigs[1].Tag = envTc.ChainConfigs[1].Tag
	}

	if os.Getenv(ChainBinaryEnv) != "" {
		for i := range fromFile.ChainConfigs {
			fromFile.ChainConfigs[i].Binary = envTc.ChainConfigs[i].Binary
		}
	}

	if os.Getenv(ChainImageEnv) != "" {
		for i := range fromFile.ChainConfigs {
			fromFile.ChainConfigs[i].Image = envTc.ChainConfigs[i].Image
		}
	}

	if os.Getenv(RelayerTagEnv) != "" {
		fromFile.RelayerConfig.Tag = envTc.RelayerConfig.Tag
	}

	if os.Getenv(RelayerTypeEnv) != "" {
		fromFile.RelayerConfig.Type = envTc.RelayerConfig.Type
	}

	if os.Getenv(ChainUpgradePlanEnv) != "" {
		fromFile.UpgradeConfig.PlanName = envTc.UpgradeConfig.PlanName
	}

	if os.Getenv(ChainUpgradeTagEnv) != "" {
		fromFile.UpgradeConfig.Tag = envTc.UpgradeConfig.Tag
	}

	return fromFile
}

// fromEnv returns a TestConfig constructed from environment variables.
func fromEnv() TestConfig {
	return TestConfig{
		ChainConfigs:  getChainConfigsFromEnv(),
		UpgradeConfig: getUpgradePlanConfigFromEnv(),
		RelayerConfig: getRelayerConfigFromEnv(),
	}
}

// getChainConfigsFromEnv returns the chain configs from environment variables.
func getChainConfigsFromEnv() []ChainConfig {
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
	return []ChainConfig{
		{
			Image:  chainAImage,
			Tag:    chainATag,
			Binary: chainBinary,
		},
		{
			Image:  chainBImage,
			Tag:    chainBTag,
			Binary: chainBinary,
		},
	}
}

// getConfigFilePath returns the absolute path where the e2e config file should be.
func getConfigFilePath() string {
	if absoluteConfigPath := os.Getenv(E2EConfigFilePathEnv); absoluteConfigPath != "" {
		return absoluteConfigPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return path.Join(homeDir, defaultConfigFileName)
}

// getRelayerConfigFromEnv returns the RelayerConfig from present environment variables.
func getRelayerConfigFromEnv() relayer.Config {
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
			rlyTag = defaultHermesTag
		}
	}
	return relayer.Config{
		Tag:  rlyTag,
		Type: relayerType,
	}
}

// getUpgradePlanConfigFromEnv returns the upgrade config from environment variables.
func getUpgradePlanConfigFromEnv() UpgradeConfig {
	upgradeTag, ok := os.LookupEnv(ChainUpgradeTagEnv)
	if !ok {
		upgradeTag = ""
	}

	upgradePlan, ok := os.LookupEnv(ChainUpgradePlanEnv)
	if !ok {
		upgradePlan = ""
	}
	return UpgradeConfig{
		PlanName: upgradePlan,
		Tag:      upgradeTag,
	}
}

func GetChainATag() string {
	return LoadConfig().ChainConfigs[0].Tag
}

func GetChainBTag() string {
	if chainBTag := LoadConfig().ChainConfigs[1].Tag; chainBTag != "" {
		return chainBTag
	}
	return GetChainATag()
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
	tc := LoadConfig()

	chainACfg := newDefaultSimappConfig(tc.ChainConfigs[0], "simapp-a", tc.GetChainAID(), "atoma", tc.CometBFTConfig)
	chainBCfg := newDefaultSimappConfig(tc.ChainConfigs[1], "simapp-b", tc.GetChainBID(), "atomb", tc.CometBFTConfig)
	return ChainOptions{
		ChainAConfig: &chainACfg,
		ChainBConfig: &chainBCfg,
	}
}

// newDefaultSimappConfig creates an ibc configuration for simd.
func newDefaultSimappConfig(cc ChainConfig, name, chainID, denom string, cometCfg CometBFTConfig) ibc.ChainConfig {
	configFileOverrides := make(map[string]any)
	tmTomlOverrides := make(interchaintestutil.Toml)

	tmTomlOverrides["log_level"] = cometCfg.LogLevel // change to debug in ~/.ibc-go-e2e-config.json to increase cometbft logging.
	configFileOverrides["config/config.toml"] = tmTomlOverrides

	var useNewGenesisCommand bool
	if cc.Binary == defaultBinary && testvalues.SimdNewGenesisCommandsFeatureReleases.IsSupported(cc.Tag) {
		useNewGenesisCommand = true
	}

	if cc.Binary == icadBinary && testvalues.IcadNewGenesisCommandsFeatureReleases.IsSupported(cc.Tag) {
		useNewGenesisCommand = true
	}

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
		Bin:                    cc.Binary,
		Bech32Prefix:           "cosmos",
		CoinType:               fmt.Sprint(sdk.GetConfig().GetCoinType()),
		Denom:                  denom,
		GasPrices:              fmt.Sprintf("0.00%s", denom),
		GasAdjustment:          1.3,
		TrustingPeriod:         "508h",
		NoHostMount:            false,
		ModifyGenesis:          getGenesisModificationFunction(cc),
		ConfigFileOverrides:    configFileOverrides,
		UsingNewGenesisCommand: useNewGenesisCommand,
	}
}

// getGenesisModificationFunction returns a genesis modification function that handles the GenesisState type
// correctly depending on if the govv1beta1 gov module is used or if govv1 is being used.
func getGenesisModificationFunction(cc ChainConfig) func(ibc.ChainConfig, []byte) ([]byte, error) {
	binary := cc.Binary
	version := cc.Tag

	simdSupportsGovV1Genesis := binary == defaultBinary && testvalues.GovGenesisFeatureReleases.IsSupported(version)
	icadSupportsGovV1Genesis := testvalues.IcadGovGenesisFeatureReleases.IsSupported(version)

	if simdSupportsGovV1Genesis || icadSupportsGovV1Genesis {
		return defaultGovv1ModifyGenesis()
	}

	return defaultGovv1Beta1ModifyGenesis()
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
