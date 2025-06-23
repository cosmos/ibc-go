package testsuite

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	interchaintestutil "github.com/cosmos/interchaintest/v10/testutil"
	"gopkg.in/yaml.v2"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	cmtjson "github.com/cometbft/cometbft/libs/json"

	"github.com/cosmos/ibc-go/e2e/internal/directories"
	"github.com/cosmos/ibc-go/e2e/relayer"
	"github.com/cosmos/ibc-go/e2e/semverutil"
	"github.com/cosmos/ibc-go/e2e/testvalues"
	wasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctypes "github.com/cosmos/ibc-go/v10/modules/core/types"
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
	// ChainCTagEnv specifies the tag that Chain C will use.
	// the value will default to the same value as Chain A.
	ChainCTagEnv = "CHAIN_C_TAG"
	// ChainDTagEnv specifies the tag that Chain D will use. If unspecified
	// the value will default to the same value as Chain A.
	ChainDTagEnv = "CHAIN_D_TAG"
	// RelayerIDEnv specifies the ID of the relayer to use.
	RelayerIDEnv = "RELAYER_ID"
	// ChainBinaryEnv binary is the binary that will be used for both chains.
	ChainBinaryEnv = "CHAIN_BINARY"
	// ChainUpgradePlanEnv specifies the upgrade plan name
	ChainUpgradePlanEnv = "CHAIN_UPGRADE_PLAN"
	// E2EConfigFilePathEnv allows you to specify a custom path for the config file to be used. It can be relative
	// or absolute.
	E2EConfigFilePathEnv = "E2E_CONFIG_PATH"
	// KeepContainersEnv instructs interchaintest to not delete the containers after a test has run.
	// this ensures that chain containers are not deleted after a test suite is run if other tests
	// depend on those chains.
	KeepContainersEnv = "KEEP_CONTAINERS"

	// defaultBinary is the default binary that will be used by the chains.
	defaultBinary = "simd"
	// defaultRlyTag is the tag that will be used if no relayer tag is specified.
	// all images are here https://github.com/cosmos/relayer/pkgs/container/relayer/versions
	defaultRlyTag = "latest"

	// defaultHermesTag is the tag that will be used if no relayer tag is specified for hermes.
	defaultHermesTag = "1.13.1"
	// defaultChainTag is the tag that will be used for the chains if none is specified.
	defaultChainTag = "main"
	// defaultConfigFileName is the default filename for the config file that can be used to configure
	// e2e tests. See sample.config.yaml or sample.config.extended.yaml as an example for what this should look like.
	defaultConfigFileName = ".ibc-go-e2e-config.yaml"
	// defaultCIConfigFileName is the default filename for the config file that should be used for CI.
	defaultCIConfigFileName = "ci-e2e-config.yaml"
)

// defaultChainNames contains the default name for chainA, chainB, ChainC and ChainD.
var defaultChainNames = []string{"simapp-a", "simapp-b", "simapp-c", "simapp-d"}

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
	// RelayerConfigs holds all known relayer configurations that can be used in the tests.
	RelayerConfigs []relayer.Config `yaml:"relayers"`
	// ActiveRelayer specifies the relayer that will be used. It must match the ID of one of the entries in RelayerConfigs.
	ActiveRelayer string `yaml:"activeRelayer"`
	// CometBFTConfig holds values for configuring CometBFT.
	CometBFTConfig CometBFTConfig `yaml:"cometbft"`
	// DebugConfig holds configuration for miscellaneous options.
	DebugConfig DebugConfig `yaml:"debug"`
	// UpgradePlanName specifies which upgrade plan to use. It must match a plan name for an entry in the
	// list of UpgradeConfigs.
	UpgradePlanName string `yaml:"upgradePlanName"`
	// UpgradeConfigs provides a list of all possible upgrades.
	UpgradeConfigs []UpgradeConfig `yaml:"upgrades"`
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

	if err := tc.validateGenesisDebugConfig(); err != nil {
		return fmt.Errorf("invalid Genesis debug configuration: %w", err)
	}

	if err := tc.validateUpgradeConfig(); err != nil {
		return fmt.Errorf("invalid upgrade configuration: %w", err)
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

		if cfg.NumValidators == 0 && cfg.NumFullNodes == 0 {
			return fmt.Errorf("chain config missing number of validators or full nodes: %+v", cfg)
		}
	}

	// clienttypes.ParseChainID is used to determine revision heights. If the chainIDs are not in the expected format,
	// tests can fail with timeout errors.
	if clienttypes.ParseChainID(tc.GetChainAID()) != clienttypes.ParseChainID(tc.GetChainBID()) {
		return fmt.Errorf("ensure both chainIDs are in the format {chainID}-{revision} and have the same revision. Got: chainA: %s, chainB: %s", tc.GetChainAID(), tc.GetChainBID())
	}

	return nil
}

// validateRelayers validates relayer configuration.
func (tc TestConfig) validateRelayers() error {
	if len(tc.RelayerConfigs) < 1 {
		return errors.New("no relayer configurations specified")
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

// GetUpgradeConfig returns the upgrade configuration for the current test configuration.
func (tc TestConfig) GetUpgradeConfig() UpgradeConfig {
	for _, upgrade := range tc.UpgradeConfigs {
		if upgrade.PlanName == tc.UpgradePlanName {
			return upgrade
		}
	}
	panic("upgrade plan not found in upgrade configs, this test config should not have passed validation")
}

// GetChainIndex returns the index of the chain with the given name, if it
// exists.
func (tc TestConfig) GetChainIndex(name string) (int, error) {
	for i := range tc.ChainConfigs {
		chainName := tc.GetChainName(i)
		if chainName == name {
			return i, nil
		}
	}
	return -1, fmt.Errorf("chain %s not found in chain configs", name)
}

// validateGenesisDebugConfig validates configuration of Genesis debug options/
func (tc TestConfig) validateGenesisDebugConfig() error {
	cfg := tc.DebugConfig.GenesisDebug
	if !cfg.DumpGenesisDebugInfo {
		return nil
	}

	// Verify that the provided chain exists in our config
	_, err := tc.GetChainIndex(tc.GetGenesisChainName())

	return err
}

// validateUpgradeConfig ensures the upgrade configuration is valid.
func (tc TestConfig) validateUpgradeConfig() error {
	if strings.TrimSpace(tc.UpgradePlanName) == "" {
		return nil
	}

	// the upgrade plan name specified must match one of the upgrade plans in the upgrade configs.
	foundPlan := false
	for _, upgrade := range tc.UpgradeConfigs {
		if strings.TrimSpace(upgrade.Tag) == "" {
			return fmt.Errorf("upgrade config missing tag: %+v", upgrade)
		}

		if strings.TrimSpace(upgrade.PlanName) == "" {
			return fmt.Errorf("upgrade config missing plan name: %+v", upgrade)
		}

		if upgrade.PlanName == tc.UpgradePlanName {
			foundPlan = true
		}
	}

	if foundPlan {
		return nil
	}

	return fmt.Errorf("upgrade plan %s not found in upgrade configs: %+v", tc.UpgradePlanName, tc.UpgradeConfigs)
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

// GetChainID returns the chain-id for i. Assumes indicies are correct.
func (tc TestConfig) GetChainID(i int) string {
	if tc.ChainConfigs[i].ChainID != "" {
		return tc.ChainConfigs[i].ChainID
	}
	return fmt.Sprintf("chain%c-1", 'A'+i)
}

// GetChainAID returns the chain-id for chain A.
// NOTE: the default return value will ensure that ParseChainID will return 1 as the revision number.
func (tc TestConfig) GetChainAID() string {
	if tc.ChainConfigs[0].ChainID != "" {
		return tc.ChainConfigs[0].ChainID
	}
	return "chainA-1"
}

// GetChainBID returns the chain-id for chain B.
// NOTE: the default return value will ensure that ParseChainID will return 1 as the revision number.
func (tc TestConfig) GetChainBID() string {
	if tc.ChainConfigs[1].ChainID != "" {
		return tc.ChainConfigs[1].ChainID
	}
	return "chainB-1"
}

// GetChainCID returns the chain-id for chain C.
// NOTE: the default return value will ensure that ParseChainID will return 1 as the revision number.
func (tc TestConfig) GetChainCID() string {
	if tc.ChainConfigs[2].ChainID != "" {
		return tc.ChainConfigs[2].ChainID
	}
	return "chainC-1"
}

// GetChainDID returns the chain-id for chain D.
// NOTE: the default return value will ensure that ParseChainID will return 1 as the revision number.
func (tc TestConfig) GetChainDID() string {
	if tc.ChainConfigs[3].ChainID != "" {
		return tc.ChainConfigs[3].ChainID
	}
	return "chainD-1"
}

// GetChainName returns the name of the chain given an index.
func (tc TestConfig) GetChainName(idx int) string {
	// Assumes that only valid indices are provided. We do the same in several other places.
	chainName := tc.ChainConfigs[idx].Name
	if chainName == "" {
		chainName = defaultChainNames[idx]
	}
	return chainName
}

// GetGenesisChainName returns the name of the chain for which to dump Genesis files.
// If no chain is provided, it uses the default one (chainA).
func (tc TestConfig) GetGenesisChainName() string {
	name := tc.DebugConfig.GenesisDebug.ChainName
	if name == "" {
		return tc.GetChainName(0)
	}
	return name
}

// UpgradeConfig holds values relevant to upgrade tests.
type UpgradeConfig struct {
	PlanName string `yaml:"planName"`
	Tag      string `yaml:"tag"`
}

// ChainConfig holds information about an individual chain used in the tests.
type ChainConfig struct {
	ChainID       string `yaml:"chainId"`
	Name          string `yaml:"name"`
	Image         string `yaml:"image"`
	Tag           string `yaml:"tag"`
	Binary        string `yaml:"binary"`
	NumValidators int    `yaml:"numValidators"`
	NumFullNodes  int    `yaml:"numFullNodes"`
}

type CometBFTConfig struct {
	LogLevel string `yaml:"logLevel"`
}

type GenesisDebugConfig struct {
	// DumpGenesisDebugInfo enables the output of Genesis debug files.
	DumpGenesisDebugInfo bool `yaml:"dumpGenesisDebugInfo"`

	// ExportFilePath specifies which path to export Genesis debug files to.
	ExportFilePath string `yaml:"filePath"`

	// ChainName represent which chain to get Genesis debug info for.
	ChainName string `yaml:"chainName"`
}

type DebugConfig struct {
	// DumpLogs forces the logs to be collected before removing test containers.
	DumpLogs bool `yaml:"dumpLogs"`

	// GenesisDebug contains debug information specific to Genesis.
	GenesisDebug GenesisDebugConfig `yaml:"genesis"`

	// KeepContainers specifies if the containers should be kept after the test suite is done.
	// NOTE: when running a full test suite, this value should be set to true in order to preserve
	// shared resources.
	KeepContainers bool `yaml:"keepContainers"`
}

// LoadConfig attempts to load a test configuration from the default file path.
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

	testCfg := applyEnvironmentVariableOverrides(fileTc)

	// If tags for chain C and D are not present in the file, also not set in the CI, fallback to A
	if testCfg.ChainConfigs[2].Tag == "" {
		testCfg.ChainConfigs[2].Tag = testCfg.ChainConfigs[0].Tag
	}
	if testCfg.ChainConfigs[3].Tag == "" {
		testCfg.ChainConfigs[3].Tag = testCfg.ChainConfigs[0].Tag
	}
	return testCfg
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

	return populateDefaults(tc), true
}

// populateDefaults populates default values for the test config if
// certain required fields are not specified.
func populateDefaults(tc TestConfig) TestConfig {
	chainIDs := []string{
		"chainA-1",
		"chainB-1",
		"chainC-1",
		"chainD-1",
	}

	for i := range tc.ChainConfigs {
		if tc.ChainConfigs[i].ChainID == "" {
			tc.ChainConfigs[i].ChainID = chainIDs[i]
		}
		if tc.ChainConfigs[i].Binary == "" {
			tc.ChainConfigs[i].Binary = defaultBinary
		}
		if tc.ChainConfigs[i].Image == "" {
			tc.ChainConfigs[i].Image = getChainImage(tc.ChainConfigs[i].Binary)
		}
		if tc.ChainConfigs[i].NumValidators == 0 {
			tc.ChainConfigs[i].NumValidators = 1
		}

		// If tag not given for chain C and D, set to chain A' tag
		if tc.ChainConfigs[i].Tag == "" && i != 0 {
			tc.ChainConfigs[i].Tag = tc.ChainConfigs[0].Tag
		}
	}

	if tc.ActiveRelayer == "" {
		tc.ActiveRelayer = relayer.Hermes
	}

	if tc.RelayerConfigs == nil {
		tc.RelayerConfigs = []relayer.Config{
			getDefaultRlyRelayerConfig(),
			getDefaultHermesRelayerConfig(),
		}
	}

	if tc.CometBFTConfig.LogLevel == "" {
		tc.CometBFTConfig.LogLevel = "info"
	}

	return tc
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

	if os.Getenv(ChainCTagEnv) != "" {
		fromFile.ChainConfigs[2].Tag = envTc.ChainConfigs[2].Tag
	}

	if os.Getenv(ChainDTagEnv) != "" {
		fromFile.ChainConfigs[3].Tag = envTc.ChainConfigs[3].Tag
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
		fromFile.UpgradePlanName = envTc.UpgradePlanName
	}

	if isEnvTrue(KeepContainersEnv) {
		fromFile.DebugConfig.KeepContainers = true
	}

	return fromFile
}

// fromEnv returns a TestConfig constructed from environment variables.
func fromEnv() TestConfig {
	return TestConfig{
		ChainConfigs:    getChainConfigsFromEnv(),
		UpgradePlanName: os.Getenv(ChainUpgradePlanEnv),
		ActiveRelayer:   os.Getenv(RelayerIDEnv),
		CometBFTConfig:  CometBFTConfig{LogLevel: "info"},
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

	chainCTag, ok := os.LookupEnv(ChainCTagEnv)
	if !ok {
		chainCTag = chainATag
	}

	chainDTag, ok := os.LookupEnv(ChainDTagEnv)
	if !ok {
		chainDTag = chainATag
	}

	chainAImage := getChainImage(chainBinary)
	specifiedChainImage, ok := os.LookupEnv(ChainImageEnv)
	if ok {
		chainAImage = specifiedChainImage
	}

	numValidators := 4
	numFullNodes := 1

	chainBImage := chainAImage
	chainCImage := chainAImage
	chainDImage := chainAImage

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
		{
			Image:         chainCImage,
			Tag:           chainCTag,
			Binary:        chainBinary,
			NumValidators: numValidators,
			NumFullNodes:  numFullNodes,
		},
		{
			Image:         chainDImage,
			Tag:           chainDTag,
			Binary:        chainBinary,
			NumValidators: numValidators,
			NumFullNodes:  numFullNodes,
		},
	}
}

// getConfigFilePath returns the absolute path where the e2e config file should be.
func getConfigFilePath() string {
	if specifiedConfigPath := os.Getenv(E2EConfigFilePathEnv); specifiedConfigPath != "" {
		if path.IsAbs(specifiedConfigPath) {
			return specifiedConfigPath
		}

		e2eDir, err := directories.E2E()
		if err != nil {
			panic(err)
		}

		return path.Join(e2eDir, specifiedConfigPath)
	}

	if IsCI() {
		if err := os.Setenv(E2EConfigFilePathEnv, defaultCIConfigFileName); err != nil {
			panic(err)
		}
		return getConfigFilePath()
	}

	// running locally.
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
	return isEnvTrue("CI")
}

// IsFork returns true if the tests are running in fork mode, false is returned otherwise.
func IsFork() bool {
	return isEnvTrue("FORK")
}

// IsRunSuite returns true if the tests are running in suite mode, false is returned otherwise.
func IsRunSuite() bool {
	return isEnvTrue("RUN_SUITE")
}

func isEnvTrue(env string) bool {
	return strings.ToLower(os.Getenv(env)) == "true"
}

// ChainOptions stores chain configurations for the chains that will be
// created for the tests. They can be modified by passing ChainOptionConfiguration
// to E2ETestSuite.GetChains.
type ChainOptions struct {
	ChainSpecs       []*interchaintest.ChainSpec
	SkipPathCreation bool
	RelayerCount     int
}

// ChainOptionConfiguration enables arbitrary configuration of ChainOptions.
type ChainOptionConfiguration func(options *ChainOptions)

// DefaultChainOptions returns the default configuration for required number of chains.
// These options can be configured by passing configuration functions to E2ETestSuite.GetChains.
func DefaultChainOptions(chainCount int) (ChainOptions, error) {
	tc := LoadConfig()

	if len(tc.ChainConfigs) < chainCount {
		return ChainOptions{}, fmt.Errorf("file has %d configs. want %d configs", len(tc.ChainConfigs), chainCount)
	}

	specs := make([]*interchaintest.ChainSpec, 0, chainCount)
	for i := range chainCount {
		denom := fmt.Sprintf("atom%c", 'a'+i)
		chainName := tc.GetChainName(i)
		chainID := tc.GetChainID(i)
		cfg := newDefaultSimappConfig(tc.ChainConfigs[0], chainName, chainID, denom, tc.CometBFTConfig)
		validators, fullNodes := getValidatorsAndFullNodes(i)

		spec := &interchaintest.ChainSpec{
			ChainConfig:   cfg,
			NumFullNodes:  &fullNodes,
			NumValidators: &validators,
		}
		specs = append(specs, spec)
	}

	// if running a single test, only one relayer is needed.
	numRelayers := 1
	if IsRunSuite() {
		// arbitrary number that will not be required if https://github.com/cosmos/interchaintest/issues/1153 is resolved.
		// It can be overridden in individual test suites in SetupSuite if required.
		numRelayers = 10
	}

	return ChainOptions{
		ChainSpecs:   specs,
		RelayerCount: numRelayers,
	}, nil
}

// newDefaultSimappConfig creates an ibc configuration for simd.
func newDefaultSimappConfig(cc ChainConfig, name, chainID, denom string, cometCfg CometBFTConfig) ibc.ChainConfig {
	configFileOverrides := make(map[string]any)
	tmTomlOverrides := make(interchaintestutil.Toml)

	tmTomlOverrides["log_level"] = cometCfg.LogLevel // change to debug in the e2e test config to increase cometbft logging.
	configFileOverrides["config/config.toml"] = tmTomlOverrides

	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    name,
		ChainID: chainID,
		Images: []ibc.DockerImage{
			{
				Repository: cc.Image,
				Version:    cc.Tag,
				UIDGID:     "1000:1000",
			},
		},
		Bin:                 cc.Binary,
		Bech32Prefix:        "cosmos",
		CoinType:            fmt.Sprint(sdk.CoinType),
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

	// TODO: Remove after we drop v7 support (this is only needed right now because of v6 -> v7 upgrade tests)
	if simdSupportsGovV1Genesis {
		return defaultGovv1ModifyGenesis(version)
	}

	return defaultGovv1Beta1ModifyGenesis(version)
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

		if !testvalues.AllowAllClientsWildcardFeatureReleases.IsSupported(version) {
			ibcGenBz, err := modifyClientGenesisAppState(appState[ibcexported.ModuleName])
			if err != nil {
				return nil, err
			}
			appState[ibcexported.ModuleName] = ibcGenBz
		}

		if !testvalues.ChannelParamsFeatureReleases.IsSupported(version) {
			ibcGenBz, err := modifyChannelGenesisAppState(appState[ibcexported.ModuleName])
			if err != nil {
				return nil, err
			}
			appState[ibcexported.ModuleName] = ibcGenBz
		}

		if !testvalues.ChannelsV2FeatureReleases.IsSupported(version) {
			ibcGenBz, err := modifyChannelV2GenesisAppState(appState[ibcexported.ModuleName])
			if err != nil {
				return nil, err
			}
			appState[ibcexported.ModuleName] = ibcGenBz
		}

		if !testvalues.ClientV2FeatureReleases.IsSupported(version) {
			ibcGenBz, err := modifyClientV2GenesisAppState(appState[ibcexported.ModuleName])
			if err != nil {
				return nil, err
			}
			appState[ibcexported.ModuleName] = ibcGenBz
		}

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
func defaultGovv1Beta1ModifyGenesis(version string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	const appStateKey = "app_state"
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		genesisDocMap := map[string]any{}
		err := json.Unmarshal(genbz, &genesisDocMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis bytes into genesis doc: %w", err)
		}

		appStateMap, ok := genesisDocMap[appStateKey].(map[string]any)
		if !ok {
			return nil, errors.New("failed to extract to app_state")
		}

		govModuleBytes, err := json.Marshal(appStateMap[govtypes.ModuleName])
		if err != nil {
			return nil, fmt.Errorf("failed to extract gov genesis bytes: %w", err)
		}

		govModuleGenesisBytes, err := modifyGovv1Beta1AppState(chainConfig, govModuleBytes)
		if err != nil {
			return nil, err
		}

		govModuleGenesisMap := map[string]any{}
		err = json.Unmarshal(govModuleGenesisBytes, &govModuleGenesisMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal gov genesis bytes into map: %w", err)
		}

		if !testvalues.AllowAllClientsWildcardFeatureReleases.IsSupported(version) {
			ibcModuleBytes, err := json.Marshal(appStateMap[ibcexported.ModuleName])
			if err != nil {
				return nil, fmt.Errorf("failed to extract ibc genesis bytes: %w", err)
			}

			ibcGenesisBytes, err := modifyClientGenesisAppState(ibcModuleBytes)
			if err != nil {
				return nil, err
			}

			ibcModuleGenesisMap := map[string]any{}
			err = json.Unmarshal(ibcGenesisBytes, &ibcModuleGenesisMap)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal gov genesis bytes into map: %w", err)
			}
			appStateMap[ibcexported.ModuleName] = ibcModuleGenesisMap
		}

		if !testvalues.ChannelParamsFeatureReleases.IsSupported(version) {
			ibcModuleBytes, err := json.Marshal(appStateMap[ibcexported.ModuleName])
			if err != nil {
				return nil, fmt.Errorf("failed to extract ibc genesis bytes: %w", err)
			}

			ibcGenesisBytes, err := modifyChannelGenesisAppState(ibcModuleBytes)
			if err != nil {
				return nil, err
			}

			ibcModuleGenesisMap := map[string]any{}
			err = json.Unmarshal(ibcGenesisBytes, &ibcModuleGenesisMap)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal gov genesis bytes into map: %w", err)
			}
			appStateMap[ibcexported.ModuleName] = ibcModuleGenesisMap
		}

		if !testvalues.ChannelsV2FeatureReleases.IsSupported(version) {
			ibcModuleBytes, err := json.Marshal(appStateMap[ibcexported.ModuleName])
			if err != nil {
				return nil, fmt.Errorf("failed to extract ibc genesis bytes: %w", err)
			}

			ibcGenesisBytes, err := modifyChannelV2GenesisAppState(ibcModuleBytes)
			if err != nil {
				return nil, err
			}

			ibcModuleGenesisMap := map[string]any{}
			err = json.Unmarshal(ibcGenesisBytes, &ibcModuleGenesisMap)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal gov genesis bytes into map: %w", err)
			}
			appStateMap[ibcexported.ModuleName] = ibcModuleGenesisMap
		}

		if !testvalues.ClientV2FeatureReleases.IsSupported(version) {
			ibcModuleBytes, err := json.Marshal(appStateMap[ibcexported.ModuleName])
			if err != nil {
				return nil, fmt.Errorf("failed to extract ibc genesis bytes: %w", err)
			}

			ibcGenesisBytes, err := modifyClientV2GenesisAppState(ibcModuleBytes)
			if err != nil {
				return nil, err
			}

			ibcModuleGenesisMap := map[string]any{}
			err = json.Unmarshal(ibcGenesisBytes, &ibcModuleGenesisMap)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal gov genesis bytes into map: %w", err)
			}
			appStateMap[ibcexported.ModuleName] = ibcModuleGenesisMap
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

// modifyClientGenesisAppState takes the existing ibc app state and marshals it to an ibc GenesisState.
func modifyClientGenesisAppState(ibcAppState []byte) ([]byte, error) {
	cfg := testutil.MakeTestEncodingConfig()

	cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)
	clienttypes.RegisterInterfaces(cfg.InterfaceRegistry)

	ibcGenesisState := &ibctypes.GenesisState{}
	if err := cdc.UnmarshalJSON(ibcAppState, ibcGenesisState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis bytes into client genesis state: %w", err)
	}

	ibcGenesisState.ClientGenesis.Params.AllowedClients = append(ibcGenesisState.ClientGenesis.Params.AllowedClients, wasmtypes.Wasm)
	ibcGenBz, err := cdc.MarshalJSON(ibcGenesisState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gov genesis state: %w", err)
	}

	return ibcGenBz, nil
}

// modifyChannelGenesisAppState takes the existing ibc app state, unmarshals it to a map and removes the `params` entry from ibc channel genesis.
// It marshals and returns the ibc GenesisState JSON map as bytes.
func modifyChannelGenesisAppState(ibcAppState []byte) ([]byte, error) {
	var ibcGenesisMap map[string]any
	if err := json.Unmarshal(ibcAppState, &ibcGenesisMap); err != nil {
		return nil, err
	}

	var channelGenesis map[string]any
	// be ashamed, be very ashamed
	channelGenesis, ok := ibcGenesisMap["channel_genesis"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("can't convert IBC genesis map entry into type %T", &channelGenesis)
	}
	delete(channelGenesis, "params")

	return json.Marshal(ibcGenesisMap)
}

func modifyChannelV2GenesisAppState(ibcAppState []byte) ([]byte, error) {
	var ibcGenesisMap map[string]any
	if err := json.Unmarshal(ibcAppState, &ibcGenesisMap); err != nil {
		return nil, err
	}
	delete(ibcGenesisMap, "channel_v2_genesis")

	return json.Marshal(ibcGenesisMap)
}

func modifyClientV2GenesisAppState(ibcAppState []byte) ([]byte, error) {
	var ibcGenesisMap map[string]any
	if err := json.Unmarshal(ibcAppState, &ibcGenesisMap); err != nil {
		return nil, err
	}
	delete(ibcGenesisMap, "client_v2_genesis")

	return json.Marshal(ibcGenesisMap)
}
