//go:build !test_e2e

package rollkit

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cosmos/ibc-go/e2e/dockerutil"
	"github.com/cosmos/ibc-go/e2e/testsuite"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	testifysuite "github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"io"
	"path"
	"sync"
	"testing"
	"time"
)

const (
	rollkitAppRepo    = "ghcr.io/chatton/rollkit"
	rollkitAppVersion = "latest"
	wasmSimappRepo    = "ghcr.io/chatton/ibc-go-wasm-simd"
	wasmSimappVersion = "latest"
	sequencerMnemonic = "clock post desk civil pottery foster expand merit dash seminar song memory figure uniform spice circle try happy obvious trash crime hybrid hood cushion"
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

// // gmd start --rollkit.aggregator --rollkit.da_address="celestia:26650" --rollkit.da_start_height $DA_BLOCK_HEIGHT --rpc.laddr tcp://0.0.0.0:36657 --grpc.address "0.0.0.0:9290" --p2p.laddr "0.0.0.0:36656" --minimum-gas-prices="0.025stake"

func (s *RollkitTestSuite) extractChainPrivateKeys(ctx context.Context, chain *cosmos.CosmosChain) PrivValidatorKeyFile {
	fr := NewFileRetriever(zap.NewNop(), s.DockerClient, s.T().Name())
	contents, err := fr.SingleFileContent(ctx, chain.Validators[0].VolumeName, "config/priv_validator_key.json")
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
			"power": "1000",
			"name":  "Rollkit Sequencer",
		},
	}

	appGenesis["consensus"] = consensusGenesis
	return json.Marshal(appGenesis)
}

func (s *RollkitTestSuite) Test_Rollkit_Succeeds() {
	_, _ = s.SetupChainsRelayerAndChannel(context.TODO(), nil, func(options *testsuite.ChainOptions) {
		options.ChainASpec.Bin = "gmd"
		options.ChainASpec.Bech32Prefix = "gm"
		options.ChainASpec.AdditionalStartArgs = []string{"--rollkit.aggregator"}

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

	chainA, chainB := s.GetChains()

	_ = chainA
	_ = chainB

	fmt.Println("sleeping!!!")
	time.Sleep(5 * time.Minute)
}

// FileRetriever allows retrieving a single file from a Docker volume.
// In the future it may allow retrieving an entire directory.
type FileRetriever struct {
	log *zap.Logger

	cli *client.Client

	testName string
}

// NewFileRetriever returns a new FileRetriever.
func NewFileRetriever(log *zap.Logger, cli *client.Client, testName string) *FileRetriever {
	return &FileRetriever{log: log, cli: cli, testName: testName}
}

// SingleFileContent returns the content of the file named at relPath,
// inside the volume specified by volumeName.
func (r *FileRetriever) SingleFileContent(ctx context.Context, volumeName, relPath string) ([]byte, error) {
	const mountPath = "/mnt/dockervolume"

	if err := ensureBusybox(ctx, r.cli); err != nil {
		return nil, err
	}

	containerName := fmt.Sprintf("interchaintest-getfile-%d-%s", time.Now().UnixNano(), dockerutil.RandLowerCaseLetterString(5))

	cc, err := r.cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: busyboxRef,

			// Use root user to avoid permission issues when reading files from the volume.
			User: dockerutil.GetRootUserString(),

			Labels: map[string]string{dockerutil.CleanupLabel: r.testName},
		},
		&container.HostConfig{
			Binds:      []string{volumeName + ":" + mountPath},
			AutoRemove: true,
		},
		nil, // No networking necessary.
		nil,
		containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("creating container: %w", err)
	}

	defer func() {
		if err := r.cli.ContainerRemove(ctx, cc.ID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			r.log.Warn("Failed to remove file content container", zap.String("container_id", cc.ID), zap.Error(err))
		}
	}()

	rc, _, err := r.cli.CopyFromContainer(ctx, cc.ID, path.Join(mountPath, relPath))
	if err != nil {
		return nil, fmt.Errorf("copying from container: %w", err)
	}
	defer func() {
		_ = rc.Close()
	}()

	wantPath := path.Base(relPath)
	tr := tar.NewReader(rc)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar from container: %w", err)
		}
		if hdr.Name != wantPath {
			r.log.Debug("Unexpected path", zap.String("want", relPath), zap.String("got", hdr.Name))
			continue
		}

		return io.ReadAll(tr)
	}

	return nil, fmt.Errorf("path %q not found in tar from container", relPath)
}

// Allow multiple goroutines to check for busybox
// by using a protected package-level variable.
//
// A mutex allows for retries upon error, if we ever need that;
// whereas a sync.Once would not be simple to retry.
var (
	ensureBusyboxMu sync.Mutex
	hasBusybox      bool
)

const busyboxRef = "busybox:stable"

func ensureBusybox(ctx context.Context, cli *client.Client) error {
	ensureBusyboxMu.Lock()
	defer ensureBusyboxMu.Unlock()

	if hasBusybox {
		return nil
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", busyboxRef)),
	})
	if err != nil {
		return fmt.Errorf("listing images to check busybox presence: %w", err)
	}

	if len(images) > 0 {
		hasBusybox = true
		return nil
	}

	rc, err := cli.ImagePull(ctx, busyboxRef, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()

	hasBusybox = true
	return nil
}
