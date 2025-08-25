package relayer

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/relayer"
	"github.com/cosmos/interchaintest/v10/relayer/hermes"
	dockerclient "github.com/moby/moby/client"
	"github.com/pelletier/go-toml"
	"go.uber.org/zap"
)

const (
	Rly    = "rly"
	Hermes = "hermes"

	HermesRelayerRepository = "ghcr.io/informalsystems/hermes"
	hermesRelayerUser       = "2000:2000"
	RlyRelayerRepository    = "ghcr.io/cosmos/relayer"
	rlyRelayerUser          = "100:1000"

	// relativeHermesConfigFilePath is the path to the hermes config file relative to the home directory within the container.
	relativeHermesConfigFilePath = ".hermes/config.toml"
)

// Config holds configuration values for the relayer used in the tests.
type Config struct {
	// Tag is the tag used for the relayer image.
	Tag string `yaml:"tag"`
	// ID specifies the type of relayer that this is.
	ID string `yaml:"id"`
	// Image is the image that should be used for the relayer.
	Image string `yaml:"image"`
}

// New returns an implementation of ibc.Relayer depending on the provided RelayerType.
func New(t *testing.T, cfg Config, logger *zap.Logger, dockerClient *dockerclient.Client, network string) ibc.Relayer {
	t.Helper()
	switch cfg.ID {
	case Rly:
		return newCosmosRelayer(t, cfg.Tag, logger, dockerClient, network, cfg.Image)
	case Hermes:
		return newHermesRelayer(t, cfg.Tag, logger, dockerClient, network, cfg.Image)
	default:
		panic(fmt.Errorf("unknown relayer specified: %s", cfg.ID))
	}
}

// ApplyPacketFilter applies a packet filter to the hermes config file, which specifies a complete set of channels
// to watch for packets.
func ApplyPacketFilter(ctx context.Context, t *testing.T, r ibc.Relayer, chainID string, channels []ibc.ChannelOutput) error {
	t.Helper()

	h, ok := r.(*hermes.Relayer)
	if !ok {
		t.Logf("relayer %T does not support packet filtering, or it has not been implemented yet.", r)
		return nil
	}

	return modifyHermesConfigFile(ctx, h, func(config map[string]any) error {
		chains, ok := config["chains"].([]map[string]any)
		if !ok {
			return errors.New("failed to get chains from hermes config")
		}
		var chain map[string]any
		for _, c := range chains {
			if c["id"] == chainID {
				chain = c
				break
			}
		}

		if chain == nil {
			return fmt.Errorf("failed to find chain with id %s", chainID)
		}

		var channelEndpoints [][]string
		for _, c := range channels {
			channelEndpoints = append(channelEndpoints, []string{c.PortID, c.ChannelID})
		}

		// [chains.packet_filter]
		//	# policy = 'allow'
		//	# list = [
		//	#   ['ica*', '*'],
		//	#   ['transfer', 'channel-0'],
		//	# ]

		// TODO(chatton): explicitly enable watching of ICA channels
		// this will ensure the ICA tests pass, but this will need to be modified to make sure
		// ICA tests will succeed in parallel.
		channelEndpoints = append(channelEndpoints, []string{"ica*", "*"})

		// we explicitly override the full list, this allows this function to provide a complete set of channels to watch.
		chain["packet_filter"] = map[string]any{
			"policy": "allow",
			"list":   channelEndpoints,
		}

		return nil
	})
}

// modifyHermesConfigFile reads the hermes config file, applies a modification function and returns an error if any.
func modifyHermesConfigFile(ctx context.Context, h *hermes.Relayer, modificationFn func(map[string]any) error) error {
	bz, err := h.ReadFileFromHomeDir(ctx, relativeHermesConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to read hermes config file: %w", err)
	}

	var config map[string]any
	if err := toml.Unmarshal(bz, &config); err != nil {
		return errors.New("failed to unmarshal hermes config bytes")
	}

	if modificationFn != nil {
		if err := modificationFn(config); err != nil {
			return fmt.Errorf("failed to modify hermes config: %w", err)
		}
	}

	bz, err = toml.Marshal(config)
	if err != nil {
		return errors.New("failed to marshal hermes config bytes")
	}

	return h.WriteFileToHomeDir(ctx, relativeHermesConfigFilePath, bz)
}

// newCosmosRelayer returns an instance of the go relayer.
// Options are used to allow for relayer version selection and specifying the default processing option.
func newCosmosRelayer(t *testing.T, tag string, logger *zap.Logger, dockerClient *dockerclient.Client, network, relayerImage string) ibc.Relayer {
	t.Helper()

	customImageOption := relayer.CustomDockerImage(relayerImage, tag, rlyRelayerUser)
	relayerProcessingOption := relayer.StartupFlags("-p", "events") // relayer processes via events

	relayerFactory := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, customImageOption, relayerProcessingOption)

	return relayerFactory.Build(
		t, dockerClient, network,
	)
}

// newHermesRelayer returns an instance of the hermes relayer.
func newHermesRelayer(t *testing.T, tag string, logger *zap.Logger, dockerClient *dockerclient.Client, network, relayerImage string) ibc.Relayer {
	t.Helper()

	customImageOption := relayer.CustomDockerImage(relayerImage, tag, hermesRelayerUser)
	relayerFactory := interchaintest.NewBuiltinRelayerFactory(ibc.Hermes, logger, customImageOption)

	return relayerFactory.Build(
		t, dockerClient, network,
	)
}

// Map is a mapping from test names to a relayer set for that test.
type Map map[string]map[ibc.Wallet]bool

// AddRelayer adds the given relayer to the relayer set for the given test name.
func (r Map) AddRelayer(testName string, ibcrelayer ibc.Wallet) {
	if _, ok := r[testName]; !ok {
		r[testName] = make(map[ibc.Wallet]bool)
	}
	r[testName][ibcrelayer] = true
}

// ContainsRelayer returns true if the given relayer is in the relayer set for the given test name.
func (r Map) ContainsRelayer(testName string, wallet ibc.Wallet) bool {
	if relayerSet, ok := r[testName]; ok {
		return relayerSet[wallet]
	}
	return false
}
