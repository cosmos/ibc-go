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
