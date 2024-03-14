//go:build !test_e2e

package rollkit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cosmos/ibc-go/e2e/dockerutil"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	apitypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	testifysuite "github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	rollkitAppRepo    = "ghcr.io/chatton/rollkit"
	rollkitAppVersion = "latest"
	wasmSimappRepo    = "ghcr.io/chatton/ibc-go-wasm-simd"
	wasmSimappVersion = "latest"
	celestiaImage     = "ghcr.io/rollkit/local-celestia-devnet:v0.12.7"
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

func (s *RollkitTestSuite) createDockerNetwork() (*dockerclient.Client, string) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		panic(fmt.Errorf("failed to create docker client: %v", err))
	}

	//Clean up docker resources at end of test.
	//s.T().Cleanup(dockerCleanup(t, cli))

	// Also eagerly clean up any leftover resources from a previous test run,
	// e.g. if the test was interrupted.
	//dockerCleanup(t, cli)()

	name := fmt.Sprintf("interchaintest-%s", dockerutil.RandLowerCaseLetterString(8))
	n, err := cli.NetworkCreate(context.TODO(), name, apitypes.NetworkCreate{
		CheckDuplicate: true,
		Labels:         map[string]string{dockerutil.CleanupLabel: s.T().Name()},
	})
	if err != nil {
		panic(fmt.Errorf("failed to create docker network: %v", err))
	}

	return cli, n.ID
}

func (s *RollkitTestSuite) createAndStartCelestiaContainer() (*dockerclient.Client, string, string) {
	cli, networkID := s.createDockerNetwork()
	cc, err := cli.ContainerCreate(
		context.TODO(),
		&container.Config{
			// https://github.com/rollkit/local-celestia-devnet/blob/main/Dockerfile
			Image:    celestiaImage,
			Hostname: "celestia",
			Labels:   map[string]string{dockerutil.CleanupLabel: s.T().Name()},
		},
		&container.HostConfig{
			PublishAllPorts: true,
			AutoRemove:      false,
			DNS:             []string{},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				networkID: {},
			},
		},
		nil,
		fmt.Sprintf("celestia-%s", dockerutil.RandLowerCaseLetterString(5)),
	)
	s.Require().NoError(err, "failed to create celestia container")

	s.Require().NoError(cli.ContainerStart(context.TODO(), cc.ID, apitypes.ContainerStartOptions{}), "failed to start celestia container")

	containerInspect, err := cli.ContainerInspect(context.TODO(), cc.ID)
	if err != nil {
		return nil, "", ""
	}

	hostPort := containerInspect.NetworkSettings.Ports["26657/tcp"][0].HostPort

	s.Require().NoError(s.waitForCelestiaToBeLive(hostPort))

	return cli, networkID, hostPort
}

type CelestiaResponse struct {
	Result CelestiaBlockResult `json:"result"`
}

type CelestiaBlockResult struct {
	Block CelestiaBlock `json:"block"`
}

type CelestiaBlock struct {
	Header CelestiaBlockHeader `json:"header"`
}

type CelestiaBlockHeader struct {
	Height string `json:"height"`
}

// waitForCelestiaToBeLive waits for the celestia container to be live and producing blocks.
func (s *RollkitTestSuite) waitForCelestiaToBeLive(hostPort string) error {
	return testutil.WaitForCondition(time.Minute*1, time.Second*5, func() (bool, error) {
		resp, err := http.Get(fmt.Sprintf("http://0.0.0.0:%s/block", hostPort))
		if err != nil {
			s.T().Logf("celestia block request failed: %v", err)
			return false, nil
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			s.T().Logf("celestia block response read failed: %v", err)
			return false, err
		}

		var celestiaResult CelestiaResponse
		if err := json.Unmarshal(body, &celestiaResult); err != nil {
			s.T().Logf("celestia block response unmarshal failed: %v", err)
			return false, err
		}

		s.T().Logf("celestia block response: %+v", celestiaResult)

		return celestiaResult.Result.Block.Header.Height != "0", nil
	})
}

// extractChainPrivateKeys extracts the private keys from the priv_validator_key.json file of a chain's node.
func (s *RollkitTestSuite) extractChainPrivateKeys(ctx context.Context, chain *cosmos.CosmosChain) PrivValidatorKeyFile {
	fr := NewFileRetriever(zap.NewNop(), s.DockerClient, s.T().Name())
	contents, err := fr.SingleFileContent(ctx, chain.GetNode().VolumeName, "config/priv_validator_key.json")
	s.Require().NoError(err)
	var privValidatorKeyFile PrivValidatorKeyFile
	s.Require().NoError(json.Unmarshal(contents, &privValidatorKeyFile))
	return privValidatorKeyFile
}

// rollkitGenesisModification modifies the genesis file of the rollkit app to include the private key of the centralized sequencer.
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
	cli, networkID, celestiaPort := s.createAndStartCelestiaContainer()

	_, _ = s.SetupChainsRelayerAndChannel(context.TODO(), nil, func(options *testsuite.ChainOptions) {
		// use existing docker cli and network so the rollkit app can talk to celestia
		options.DockerNetwork = networkID
		options.DockerClient = cli

		options.ChainASpec.ChainName = "rollkit"
		options.ChainASpec.ChainID = "rollkit-app"
		options.ChainASpec.Bin = "gmd"
		options.ChainASpec.Bech32Prefix = "gm"
		// gmd start --rollkit.aggregator --rollkit.da_address="celestia:26650" --rollkit.da_start_height $DA_BLOCK_HEIGHT --rpc.laddr tcp://0.0.0.0:36657 --grpc.address "0.0.0.0:9290" --p2p.laddr "0.0.0.0:36656" --minimum-gas-prices="0.025stake"
		// options.ChainASpec.AdditionalStartArgs = []string{"--rollkit.aggregator", "--rollkit.da_address=celestia:26650", "--rpc.laddr", "tcp://0.0.0.0:36657", "--grpc.address", "0.0.0.0:9290", "--p2p.laddr", "0.0.0.0:36656", "--minimum-gas-prices=0.025atoma"}
		options.ChainASpec.AdditionalStartArgs = []string{"--rollkit.aggregator", fmt.Sprintf("--rollkit.da_address=celestia:%s", celestiaPort), "--rpc.laddr", "tcp://0.0.0.0:36657", "--grpc.address", "0.0.0.0:9290", "--p2p.laddr", "0.0.0.0:36656", "--minimum-gas-prices=0.025atoma"}

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
