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
	// these images are being develepod in the the gm-demo repo
	// https://github.com/chatton/gm-demo
	rollkitAppRepo    = "ghcr.io/chatton/rollkit"
	rollkitAppVersion = "latest"
	wasmSimappRepo    = "ghcr.io/chatton/ibc-go-wasm-simd"
	wasmSimappVersion = "latest"

	// this image is the one used in the rollkit demo.
	celestiaImage = "ghcr.io/rollkit/local-celestia-devnet:v0.12.7"
)

func TestRollkitTestSuite(t *testing.T) {
	testifysuite.Run(t, new(RollkitTestSuite))
}

type RollkitTestSuite struct {
	testsuite.E2ETestSuite
}

func (s *RollkitTestSuite) Test_Rollkit_Succeeds() {
	cli, networkID, celestiaHostPort := s.createAndStartCelestiaContainer()
	_, _ = s.SetupChainsRelayerAndChannel(context.TODO(), nil, func(options *testsuite.ChainOptions) {
		// use existing docker cli and network so the rollkit app can talk to celestia
		options.DockerNetwork = networkID
		options.DockerClient = cli

		// TODO: this is not being propagated correctly.
		// The test will currently fail when the relayer attempts to connect to the rollkit app.
		options.SkipPathCreation = true

		options.ChainASpec.ChainName = "rollkit"
		options.ChainASpec.ChainID = "rollkit-app"
		options.ChainASpec.Bin = "gmd"
		options.ChainASpec.Bech32Prefix = "gm"
		options.ChainASpec.AdditionalStartArgs = []string{
			"--rollkit.aggregator",
			fmt.Sprintf("--rollkit.da_address=celestia:26650"),
			fmt.Sprintf("--rollkit.da_start_height=%s", s.getCelestiaBlockHeight(celestiaHostPort)),
			"--minimum-gas-prices=0.025atoma",

			// NOTE: the following feels shouldn't be required, but the rollkit app fails to communicate with
			// celestia without them, the interchaintest test itself fails as the ports are not what is expected,
			// but with these values the rollkit app successfully publishes blobs to celestia.
			// ¯\_(ツ)_/¯
			"--rpc.laddr",
			"tcp://0.0.0.0:36657",
			"--grpc.address",
			"0.0.0.0:9290",
			"--p2p.laddr",
			"0.0.0.0:36656",
		}

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

// NOTE: this is mostly a copy paste of the existing code in interchain test which is unexported at the moment.
func (s *RollkitTestSuite) createDockerNetwork() (*dockerclient.Client, string) {
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		panic(fmt.Errorf("failed to create docker client: %v", err))
	}

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

// createAndStartCelestiaContainer creates and starts a celestia container and waits for it to be live.
// It returns the docker client, the network ID, the host port, and the height of the first block produced which can
// be used as the start height for the rollkit app.
func (s *RollkitTestSuite) createAndStartCelestiaContainer() (*dockerclient.Client, string, string) {
	cli, networkID := s.createDockerNetwork()
	cc, err := cli.ContainerCreate(
		context.TODO(),
		&container.Config{
			// https://github.com/rollkit/local-celestia-devnet/blob/main/Dockerfile
			Image:    celestiaImage,
			Hostname: "celestia",
			Labels:   map[string]string{dockerutil.CleanupLabel: s.T().Name()},

			// wait until celestia container is producing blocks.
			Healthcheck: &container.HealthConfig{
				Test:        []string{"curl", "http://celestia:26657/block"},
				Interval:    5 * time.Second,
				Timeout:     10 * time.Second,
				StartPeriod: time.Second,
				Retries:     10,
			},
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

	// this is the port that the rollkit app will use to connect to celestia
	hostPort := containerInspect.NetworkSettings.Ports["26657/tcp"][0].HostPort

	s.Require().NoError(s.waitForCelestiaToBeLive(hostPort))

	return cli, networkID, hostPort
}

// waitForCelestiaToBeLive waits for the celestia container to be live and producing blocks.
func (s *RollkitTestSuite) waitForCelestiaToBeLive(hostPort string) error {
	return testutil.WaitForCondition(time.Minute*1, time.Second*5, func() (bool, error) {
		return s.getCelestiaBlockHeight(hostPort) != "", nil
	})
}

func (s *RollkitTestSuite) getCelestiaBlockHeight(hostPort string) string {
	resp, err := http.Get(fmt.Sprintf("http://0.0.0.0:%s/block", hostPort))
	if err != nil {
		s.T().Logf("celestia block request failed: %v", err)
		return ""
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.T().Logf("celestia block response read failed: %v", err)
		return ""
	}

	var celestiaResult CelestiaResponse
	if err := json.Unmarshal(body, &celestiaResult); err != nil {
		s.T().Logf("celestia block response unmarshal failed: %v", err)
		return ""
	}
	return celestiaResult.Result.Block.Header.Height
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

	// the validator set must be populated at genesis and have a value of at least 1.
	// we can use the keys from the keys generated by interchaintest.
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
