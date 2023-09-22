package relayer

import (
	"fmt"
	"testing"

	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/relayer"
	"go.uber.org/zap"
)

const (
	Rly    = "rly"
	Hermes = "hermes"

	HermesRelayerRepository = "colinaxner/hermes"
	hermesRelayerUser       = "1000:1000"
	RlyRelayerRepository    = "ghcr.io/cosmos/relayer"
	rlyRelayerUser          = "100:1000" // docker run -it --rm --entrypoint echo ghcr.io/cosmos/relayer "$(id -u):$(id -g)"
)

// Config holds configuration values for the relayer used in the tests.
type Config struct {
	// Tag is the tag used for the relayer image.
	Tag string `yaml:"tag"`
	// Type specifies the type of relayer that this is.
	Type string `yaml:"type"`
	// Image is the image that should be used for the relayer.
	Image string `yaml:"image"`
}

// New returns an implementation of ibc.Relayer depending on the provided RelayerType.
func New(t *testing.T, cfg Config, logger *zap.Logger, dockerClient *dockerclient.Client, network string) ibc.Relayer {
	t.Helper()
	switch cfg.Type {
	case Rly:
		return newCosmosRelayer(t, cfg.Tag, logger, dockerClient, network, cfg.Image)
	case Hermes:
		return newHermesRelayer(t, cfg.Tag, logger, dockerClient, network, cfg.Image)
	default:
		panic(fmt.Sprintf("unknown relayer specified: %s", cfg.Type))
	}
}

// newCosmosRelayer returns an instance of the go relayer.
// Options are used to allow for relayer version selection and specifying the default processing option.
func newCosmosRelayer(t *testing.T, tag string, logger *zap.Logger, dockerClient *dockerclient.Client, network, relayerImage string) ibc.Relayer {
	t.Helper()

	if relayerImage == "" {
		relayerImage = RlyRelayerRepository
	}

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

	if relayerImage == "" {
		relayerImage = HermesRelayerRepository
	}

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

// containsRelayer returns true if the given relayer is in the relayer set for the given test name.
func (r Map) ContainsRelayer(testName string, wallet ibc.Wallet) bool {
	if relayerSet, ok := r[testName]; ok {
		return relayerSet[wallet]
	}
	return false
}
