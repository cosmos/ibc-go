package testconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"

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
	// GoRelayerTagEnv specifies the go relayer version. Defaults to "main"
	GoRelayerTagEnv = "RLY_TAG"
	// ChainBinaryEnv binary is the binary that will be used for both chains.
	ChainBinaryEnv = "CHAIN_BINARY"
	// ChainUpgradeTagEnv specifies the upgrade version tag
	ChainUpgradeTagEnv = "CHAIN_UPGRADE_TAG"
	// defaultBinary is the default binary that will be used by the chains.
	defaultBinary = "simd"
	// defaultRlyTag is the tag that will be used if no relayer tag is specified.
	// all images are here https://github.com/cosmos/relayer/pkgs/container/relayer/versions
	defaultRlyTag = "v2.2.0-rc2"
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

	// TODO: remove hard coded value
	rlyTag = "andrew-tendermint_v0.37"

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
		ModifyGenesis:  defaultModifyGenesis(),
	}
}

// govGenesisFeatureReleases represents the releases the governance module genesis
// was upgraded from v1beta1 to v1.
var govGenesisFeatureReleases = semverutil.FeatureReleases{
	MajorVersion: "v7",
}

// defaultModifyGenesis will only modify governance params to ensure the voting period and minimum deposit
// are functional for e2e testing purposes.
// Note: this function intentionally does not use the type defined here https://github.com/tendermint/tendermint/blob/v0.37.0-rc2/types/genesis.go#L38-L46
// and uses a map[string]interface{} instead.
// This approach prevents the field block.TimeIotaMs from being lost which happened when using the GenesisDoc type from tendermint version v0.37.0.
// ibctest performs the following steps when creating the genesis.json file for chains.
// - 1. Let the chain binary create its own genesis file.
// - 2. Apply any provided functions to modify the bytes of the file.
// - 3. Overwrite the file with the new contents.
// This is a problem because when the tendermint types change, marshalling into the type will cause us to lose
// values if the types have changed in between the version of the chain in the test and the version of tendermint
// imported by the e2e tests.
// By using a raw map[string]interface{} we preserve the values unknown to the e2e tests and can still change
// the values we care about.
// TODO: handle these genesis modifications in a way which is type safe and does not require us to rely on
// map[string]interface{}
func defaultModifyGenesis() func(ibc.ChainConfig, []byte) ([]byte, error) {
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

		govModuleGenesisBytes, err := modifyGovAppState(chainConfig, govModuleBytes)
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

// modifyGovAppState takes the existing gov app state and marshals it to either a govv1 GenesisState
// or a govv1beta1 GenesisState depending on the simapp version.
func modifyGovAppState(chainConfig ibc.ChainConfig, govAppState []byte) ([]byte, error) {
	cfg := testutil.MakeTestEncodingConfig()

	cdc := codec.NewProtoCodec(cfg.InterfaceRegistry)
	govv1.RegisterInterfaces(cfg.InterfaceRegistry)
	govv1beta1.RegisterInterfaces(cfg.InterfaceRegistry)

	shouldUseGovV1 := govGenesisFeatureReleases.IsSupported(chainConfig.Images[0].Version)

	var govGenesisState gogoproto.Message
	if shouldUseGovV1 {
		govGenesisState = &govv1.GenesisState{}
	} else {
		govGenesisState = &govv1beta1.GenesisState{}
	}

	if err := cdc.UnmarshalJSON(govAppState, govGenesisState); err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis bytes into gov genesis state: %w", err)
	}

	switch v := govGenesisState.(type) {
	case *govv1.GenesisState:
		if v.Params == nil {
			v.Params = &govv1.Params{}
		}
		// set correct minimum deposit using configured denom
		v.Params.MinDeposit = sdk.NewCoins(sdk.NewCoin(chainConfig.Denom, govv1beta1.DefaultMinDepositTokens))
		vp := testvalues.VotingPeriod
		v.Params.VotingPeriod = &vp
	case *govv1beta1.GenesisState:
		v.DepositParams.MinDeposit = sdk.NewCoins(sdk.NewCoin(chainConfig.Denom, govv1beta1.DefaultMinDepositTokens))
		v.VotingParams.VotingPeriod = testvalues.VotingPeriod
	}
	govGenBz, err := cdc.MarshalJSON(govGenesisState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gov genesis state: %w", err)
	}

	return govGenBz, nil
}
