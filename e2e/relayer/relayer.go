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

	cosmosRelayerRepository = "damiannolan/rly" //"ghcr.io/cosmos/relayer"
	cosmosRelayerUser       = "100:1000"        // docker run -it --rm --entrypoint echo ghcr.io/cosmos/relayer "$(id -u):$(id -g)"
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
	switch cfg.Type {
	case Rly:
		return newCosmosRelayer(t, cfg.Tag, logger, dockerClient, network)
	case Hermes:
		return newHermesRelayer()
	default:
		panic(fmt.Sprintf("unknown relayer specified: %s", cfg.Type))
	}
}

// newCosmosRelayer returns an instance of the go relayer.
// Options are used to allow for relayer version selection and specifying the default processing option.
func newCosmosRelayer(t *testing.T, tag string, logger *zap.Logger, dockerClient *dockerclient.Client, network string) ibc.Relayer {
	customImageOption := relayer.CustomDockerImage(cosmosRelayerRepository, tag, cosmosRelayerUser)
	relayerProcessingOption := relayer.StartupFlags("-p", "events") // relayer processes via events

	relayerFactory := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, customImageOption, relayerProcessingOption)

	return relayerFactory.Build(
		t, dockerClient, network,
	)
}

// newHermesRelayer returns an instance of the hermes relayer.
func newHermesRelayer() ibc.Relayer {
	panic("hermes relayer not yet implemented for interchaintest")
}

// RelayerMap is a mapping from test names to a relayer set for that test.
type RelayerMap map[string]map[ibc.Wallet]bool

// AddRelayer adds the given relayer to the relayer set for the given test name.
func (r RelayerMap) AddRelayer(testName string, relayer ibc.Wallet) {
	if _, ok := r[testName]; !ok {
		r[testName] = make(map[ibc.Wallet]bool)
	}
	r[testName][relayer] = true
}

// containsRelayer returns true if the given relayer is in the relayer set for the given test name.
func (r RelayerMap) ContainsRelayer(testName string, wallet ibc.Wallet) bool {
	if relayerSet, ok := r[testName]; ok {
		return relayerSet[wallet]
	}
	return false
}
