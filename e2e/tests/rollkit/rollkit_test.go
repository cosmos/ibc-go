//go:build !test_e2e

package rollkit

import (
	"context"
	"encoding/json"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	testifysuite "github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"testing"
)

const (
	rollkitAppRepo    = "ghcr.io/chatton/rollkit"
	rollkitAppVersion = "latest"
	wasmSimappRepo    = "ghcr.io/chatton/ibc-go-wasm-simd"
	wasmSimappVersion = "latest"
)

func TestRollkitTestSuite(t *testing.T) {
	testifysuite.Run(t, new(RollkitTestSuite))
}

type RollkitTestSuite struct {
	testsuite.E2ETestSuite
}

type PrivValidatorKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type PrivValidatorKeyFile struct {
	Address string           `json:"address"`
	PubKey  PrivValidatorKey `json:"pub_key"`
	PrivKey PrivValidatorKey `json:"priv_key"`
}

func (s *RollkitTestSuite) extractChainPrivateKeys(ctx context.Context, chain *cosmos.CosmosChain) PrivValidatorKeyFile {
	fr := NewFileRetriever(zap.NewNop(), s.DockerClient, s.T().Name())
	contents, err := fr.SingleFileContent(ctx, chain.GetNode().VolumeName, "config/priv_validator_key.json")
	s.Require().NoError(err)
	var privValidatorKeyFile PrivValidatorKeyFile
	s.Require().NoError(json.Unmarshal(contents, &privValidatorKeyFile))
	return privValidatorKeyFile
}

func (s *RollkitTestSuite) rollkitGenesisModification(config ibc.ChainConfig, genbz []byte) ([]byte, error) {
	chainA, _ := s.GetChains()

	rollkitChain := chainA.(*cosmos.CosmosChain)

	appGenesis := map[string]interface{}{}
	err := json.Unmarshal(genbz, &appGenesis)
	if err != nil {
		return nil, err
	}

	privateKeys := s.extractChainPrivateKeys(context.TODO(), rollkitChain)

	consensusGenesis := appGenesis["consensus"].(map[string]interface{})
	consensusGenesis["validators"] = []map[string]interface{}{
		{
			"address": privateKeys.Address,
			"pub_key": map[string]string{
				"type":  privateKeys.PubKey.Type,
				"value": privateKeys.PubKey.Value,
			},

			"power": "5000000", // interchaintest hard codes this value (somewhere)
			"name":  "Rollkit Sequencer",
		},
	}

	appGenesis["consensus"] = consensusGenesis
	return json.Marshal(appGenesis)
}

func (s *RollkitTestSuite) Test_Rollkit_Succeeds() {
	_, _ = s.SetupChainsRelayerAndChannel(context.TODO(), nil, func(options *testsuite.ChainOptions) {
		options.ChainASpec.ChainName = "rollkit"
		options.ChainASpec.ChainID = "rollkit-app"
		options.ChainASpec.Bin = "gmd"
		options.ChainASpec.Bech32Prefix = "gm"
		//options.ChainASpec.AdditionalStartArgs = []string{"--rollkit.aggregator"}

		options.ChainASpec.ModifyGenesis = s.rollkitGenesisModification

		// must have exactly one validator, the centralized sequencer.
		nf := 0
		nv := 1
		options.ChainASpec.NumFullNodes = &nf
		options.ChainASpec.NumValidators = &nv
		options.ChainASpec.Images[0].Repository = rollkitAppRepo
		options.ChainASpec.Images[0].Version = rollkitAppVersion

		options.ChainBSpec.Images[0].Repository = wasmSimappRepo
		options.ChainBSpec.Images[0].Version = wasmSimappVersion
	})
}
