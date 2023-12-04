package testsuite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	interchaintestutil "github.com/strangelove-ventures/interchaintest/v8/testutil"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	cmtjson "github.com/cometbft/cometbft/libs/json"

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
	// RelayerIDEnv specifies the ID of the relayer to use.
	RelayerIDEnv = "RELAYER_ID"
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
	defaultRlyTag = "latest"

	// TODO: https://github.com/cosmos/ibc-go/issues/4965
	defaultHyperspaceTag = "timeout"
	// defaultHermesTag is the tag that will be used if no relayer tag is specified for hermes.
	defaultHermesTag = "v1.7.0"
	// defaultChainTag is the tag that will be used for the chains if none is specified.
	defaultChainTag = "main"
	// defaultConfigFileName is the default filename for the config file that can be used to configure
	// e2e tests. See sample.config.yaml as an example for what this should look like.
	defaultConfigFileName = ".ibc-go-e2e-config.yaml"
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
	// RelayerConfig holds all known relayer configurations that can be used in the tests.
	RelayerConfigs []relayer.Config `yaml:"relayers"`
	// ActiveRelayer specifies the relayer that will be used. It must match the ID of one of the entries in RelayerConfigs.
	ActiveRelayer string `yaml:"activeRelayer"`
	// UpgradeConfig holds values used only for the upgrade tests.
	UpgradeConfig UpgradeConfig `yaml:"upgrade"`
	// CometBFTConfig holds values for configuring CometBFT.
	CometBFTConfig CometBFTConfig `yaml:"cometbft"`
	// DebugConfig holds configuration for miscellaneous options.
	DebugConfig DebugConfig `yaml:"debug"`
}

// Validate validates the test configuration is valid for use within the tests.
// this should be called before using the configuration.
func (tc TestConfig) Validate() error {
	if err := tc.validateChains(); err != nil {
		return fmt.Errorf("invalid chain configuration: %w", err)
	}

	if err := tc.validateRelayers(); err != nil {
		return fmt.Errorf("invalid relayer configuration: %w", err)
	}
	return nil
}

// validateChains validates the chain configurations.
func (tc TestConfig) validateChains() error {
	for _, cfg := range tc.ChainConfigs {
		if cfg.Binary == "" {
			return fmt.Errorf("chain config missing binary: %+v", cfg)
		}
		if cfg.Image == "" {
			return fmt.Errorf("chain config missing image: %+v", cfg)
		}
		if cfg.Tag == "" {
			return fmt.Errorf("chain config missing tag: %+v", cfg)
		}

		// TODO: validate chainID in https://github.com/cosmos/ibc-go/issues/4697
		// these are not passed in the CI at the moment. Defaults are used.
		if !IsCI() {
			if cfg.ChainID == "" {
				return fmt.Errorf("chain config missing chainID: %+v", cfg)
			}
		}

		// TODO: validate number of nodes in https://github.com/cosmos/ibc-go/issues/4697
		// these are not passed in the CI at the moment.
		if !IsCI() {
			if cfg.NumValidators == 0 && cfg.NumFullNodes == 0 {
				return fmt.Errorf("chain config missing number of validators or full nodes: %+v", cfg)
			}
		}
	}
	return nil
}

// validateRelayers validates relayer configuration.
func (tc TestConfig) validateRelayers() error {
	if len(tc.RelayerConfigs) < 1 {
		return fmt.Errorf("no relayer configurations specified")
	}

	for _, r := range tc.RelayerConfigs {
		if r.ID == "" {
			return fmt.Errorf("relayer config missing ID: %+v", r)
		}
		if r.Image == "" {
			return fmt.Errorf("relayer config missing image: %+v", r)
		}
		if r.Tag == "" {
			return fmt.Errorf("relayer config missing tag: %+v", r)
		}
	}

	if tc.GetActiveRelayerConfig() == nil {
		return fmt.Errorf("active relayer %s not found in relayer configs: %+v", tc.ActiveRelayer, tc.RelayerConfigs)
	}

	return nil
}

// GetActiveRelayerConfig returns the currently specified relayer config.
func (tc TestConfig) GetActiveRelayerConfig() *relayer.Config {
	for _, r := range tc.RelayerConfigs {
		if r.ID == tc.ActiveRelayer {
			return &r
		}
	}
	return nil
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
	return "chainA-1"
}

// GetChainBID returns the chain-id for chain B.
func (tc TestConfig) GetChainBID() string {
	if tc.ChainConfigs[1].ChainID != "" {
		return tc.ChainConfigs[1].ChainID
	}
	return "chainB-1"
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
	tc := getConfig()
	if err := tc.Validate(); err != nil {
		panic(err)
	}
	return tc
}

// getConfig returns the TestConfig with any environment variable overrides.
func getConfig() TestConfig {
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

	if os.Getenv(RelayerIDEnv) != "" {
		fromFile.ActiveRelayer = envTc.ActiveRelayer
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
		ActiveRelayer: os.Getenv(RelayerIDEnv),

		// TODO: we can remove this, and specify these values in a config file for the CI
		// in https://github.com/cosmos/ibc-go/issues/4697
		RelayerConfigs: []relayer.Config{
			getDefaultRlyRelayerConfig(),
			getDefaultHermesRelayerConfig(),
			getDefaultHyperspaceRelayerConfig(),
		},
		CometBFTConfig: CometBFTConfig{LogLevel: "info"},
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

	numValidators := 4
	numFullNodes := 1

	chainBImage := chainAImage
	return []ChainConfig{
		{
			Image:         chainAImage,
			Tag:           chainATag,
			Binary:        chainBinary,
			NumValidators: numValidators,
			NumFullNodes:  numFullNodes,
		},
		{
			Image:         chainBImage,
			Tag:           chainBTag,
			Binary:        chainBinary,
			NumValidators: numValidators,
			NumFullNodes:  numFullNodes,
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

// TODO: remove in https://github.com/cosmos/ibc-go/issues/4697
// getDefaultHermesRelayerConfig returns the default config for the hermes relayer.
func getDefaultHermesRelayerConfig() relayer.Config {
	return relayer.Config{
		Tag:   defaultHermesTag,
		ID:    relayer.Hermes,
		Image: relayer.HermesRelayerRepository,
	}
}

// TODO: remove in https://github.com/cosmos/ibc-go/issues/4697
// getDefaultRlyRelayerConfig returns the default config for the golang relayer.
func getDefaultRlyRelayerConfig() relayer.Config {
	return relayer.Config{
		Tag:   defaultRlyTag,
		ID:    relayer.Rly,
		Image: relayer.RlyRelayerRepository,
	}
}

// TODO: remove in https://github.com/cosmos/ibc-go/issues/4697
// getDefaultHyperspaceRelayerConfig returns the default config for the hyperspace relayer.
func getDefaultHyperspaceRelayerConfig() relayer.Config {
	return relayer.Config{
		Tag:   defaultHyperspaceTag,
		ID:    relayer.Hyperspace,
		Image: relayer.HyperspaceRelayerRepository,
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

// IsFork returns true if the tests are running in fork mode, false is returned otherwise.
func IsFork() bool {
	return strings.ToLower(os.Getenv("FORK")) == "true"
}

// ChainOptions stores chain configurations for the chains that will be
// created for the tests. They can be modified by passing ChainOptionConfiguration
// to E2ETestSuite.GetChains.
type ChainOptions struct {
	ChainASpec       *interchaintest.ChainSpec
	ChainBSpec       *interchaintest.ChainSpec
	SkipPathCreation bool
}

// ChainOptionConfiguration enables arbitrary configuration of ChainOptions.
type ChainOptionConfiguration func(options *ChainOptions)

// DefaultChainOptions returns the default configuration for the chains.
// These options can be configured by passing configuration functions to E2ETestSuite.GetChains.
func DefaultChainOptions() ChainOptions {
	tc := LoadConfig()

	chainACfg := newDefaultSimappConfig(tc.ChainConfigs[0], "simapp-a", tc.GetChainAID(), "atoma", tc.CometBFTConfig)
	chainBCfg := newDefaultSimappConfig(tc.ChainConfigs[1], "simapp-b", tc.GetChainBID(), "atomb", tc.CometBFTConfig)

	chainAVal, chainAFn := getValidatorsAndFullNodes(0)
	chainBVal, chainBFn := getValidatorsAndFullNodes(1)

	return ChainOptions{
		ChainASpec: &interchaintest.ChainSpec{
			ChainConfig:   chainACfg,
			NumFullNodes:  &chainAFn,
			NumValidators: &chainAVal,
		},
		ChainBSpec: &interchaintest.ChainSpec{
			ChainConfig:   chainBCfg,
			NumFullNodes:  &chainBFn,
			NumValidators: &chainBVal,
		},
	}
}

// newDefaultSimappConfig creates an ibc configuration for simd.
func newDefaultSimappConfig(cc ChainConfig, name, chainID, denom string, cometCfg CometBFTConfig) ibc.ChainConfig {
	configFileOverrides := make(map[string]any)
	tmTomlOverrides := make(interchaintestutil.Toml)

	tmTomlOverrides["log_level"] = cometCfg.LogLevel // change to debug in ~/.ibc-go-e2e-config.json to increase cometbft logging.
	configFileOverrides["config/config.toml"] = tmTomlOverrides

	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    name,
		ChainID: chainID,
		Images: []ibc.DockerImage{
			{
				Repository: cc.Image,
				Version:    cc.Tag,
				UidGid:     "1000:1000",
			},
		},
		Bin:                 cc.Binary,
		Bech32Prefix:        "cosmos",
		CoinType:            fmt.Sprint(sdk.GetConfig().GetCoinType()),
		Denom:               denom,
		EncodingConfig:      SDKEncodingConfig(),
		GasPrices:           fmt.Sprintf("0.00%s", denom),
		GasAdjustment:       1.3,
		TrustingPeriod:      "508h",
		NoHostMount:         false,
		ModifyGenesis:       getGenesisModificationFunction(cc),
		ConfigFileOverrides: configFileOverrides,
	}
}

// getGenesisModificationFunction returns a genesis modification function that handles the GenesisState type
// correctly depending on if the govv1beta1 gov module is used or if govv1 is being used.
func getGenesisModificationFunction(cc ChainConfig) func(ibc.ChainConfig, []byte) ([]byte, error) {
	binary := cc.Binary
	version := cc.Tag

	simdSupportsGovV1Genesis := binary == defaultBinary && testvalues.GovGenesisFeatureReleases.IsSupported(version)

	if simdSupportsGovV1Genesis {
		return defaultGovv1ModifyGenesis(version)
	}

	return defaultGovv1Beta1ModifyGenesis()
}

// defaultGovv1ModifyGenesis will only modify governance params to ensure the voting period and minimum deposit
// are functional for e2e testing purposes.
func defaultGovv1ModifyGenesis(version string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	stdlibJSONMarshalling := semverutil.FeatureReleases{MajorVersion: "v8"}
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		appGenesis, err := genutiltypes.AppGenesisFromReader(bytes.NewReader(genbz))
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into genesis doc: %w", err)
		}

		var appState genutiltypes.AppMap
		if err := json.Unmarshal(appGenesis.AppState, &appState); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into app state: %w", err)
		}

		govGenBz, err := modifyGovV1AppState(chainConfig, appState[govtypes.ModuleName])
		if err != nil {
			return nil, err
		}

		appState[govtypes.ModuleName] = govGenBz

		appGenesis.AppState, err = json.Marshal(appState)
		if err != nil {
			return nil, err
		}

		// in older version < v8, tmjson marshal must be used.
		// regular json marshalling must be used for v8 and above as the
		// sdk is de-coupled from comet.
		marshalIndentFn := cmtjson.MarshalIndent
		if stdlibJSONMarshalling.IsSupported(version) {
			marshalIndentFn = json.MarshalIndent
		}

		bz, err := marshalIndentFn(appGenesis, "", "  ")
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

// modifyGovV1AppState takes the existing gov app state and marshals it to a govv1 GenesisState.
func modifyGovV1AppState(chainConfig ibc.ChainConfig, govAppState []byte) ([]byte, error) {
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
	maxDep := time.Second * 10
	govGenesisState.Params.MaxDepositPeriod = &maxDep
	vp := testvalues.VotingPeriod
	govGenesisState.Params.VotingPeriod = &vp

	govGenBz := MustProtoMarshalJSON(govGenesisState)

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
