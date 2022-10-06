package testsuite

import (
	"testing"

	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/ibctest/v6"
	"github.com/strangelove-ventures/ibctest/v6/ibc"
	"github.com/strangelove-ventures/ibctest/v6/relayer"
	"go.uber.org/zap"

	"github.com/cosmos/ibc-go/e2e/testconfig"
)

const (
	cosmosRelayerRepository = "ghcr.io/cosmos/relayer"
	cosmosRelayerUser       = "100:1000" // docker run -it --rm --entrypoint echo ghcr.io/cosmos/relayer "$(id -u):$(id -g)"
)

// newCosmosRelayer returns an instance of the go relayer.
// Options are used to allow for relayer version selection and specifying the default processing option.
func newCosmosRelayer(t *testing.T, tc testconfig.TestConfig, logger *zap.Logger, dockerClient *dockerclient.Client, network string) ibc.Relayer {
<<<<<<< HEAD
	return ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, relayer.CustomDockerImage(cosmosRelayerRepository, tc.RlyTag)).Build(
=======
	customImageOption := relayer.CustomDockerImage(cosmosRelayerRepository, tc.RlyTag, cosmosRelayerUser)
	relayerProcessingOption := relayer.StartupFlags("-p", "events") // relayer processes via events

	relayerFactory := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, customImageOption, relayerProcessingOption)

	return relayerFactory.Build(
>>>>>>> 4bd05c6 (chore: bump ibctest version and ibc-go version to v6 for e2e module (#2479))
		t, dockerClient, network,
	)
}
