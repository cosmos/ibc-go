package relayer

import (
	"fmt"
	"testing"

	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/ibctest/v6"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/relayer"
	"go.uber.org/zap"

	"github.com/cosmos/ibc-go/e2e/testconfig"
)

const (
	Cosmos = "COSMOS"
	Hermes = "HERMES"

	cosmosRelayerRepository = "ghcr.io/cosmos/relayer"
	cosmosRelayerUser       = "100:1000" // docker run -it --rm --entrypoint echo ghcr.io/cosmos/relayer "$(id -u):$(id -g)"
)

// New returns an implementation of ibc.Relayer depending on the provided RelayerType.
func New(t *testing.T, tc testconfig.TestConfig, logger *zap.Logger, dockerClient *dockerclient.Client, network string) ibc.Relayer {
	switch tc.RelayerConfig.Type {
	case Cosmos:
		return newCosmosRelayer(t, tc, logger, dockerClient, network)
	case Hermes:
		return newHermesRelayer()
	default:
		panic(fmt.Sprintf("unknown relayer specified: %s", tc.RelayerConfig.Type))
	}
}

// newCosmosRelayer returns an instance of the go relayer.
// Options are used to allow for relayer version selection and specifying the default processing option.
func newCosmosRelayer(t *testing.T, tc testconfig.TestConfig, logger *zap.Logger, dockerClient *dockerclient.Client, network string) ibc.Relayer {
	customImageOption := relayer.CustomDockerImage(cosmosRelayerRepository, tc.RelayerConfig.Tag, cosmosRelayerUser)
	relayerProcessingOption := relayer.StartupFlags("-p", "events") // relayer processes via events

	relayerFactory := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, customImageOption, relayerProcessingOption)

	return relayerFactory.Build(
		t, dockerClient, network,
	)
}

// newHermesRelayer returns an instance of the hermes relayer.
func newHermesRelayer() ibc.Relayer {
	panic("hermes relayer not yet implemented for ibctest")
}
