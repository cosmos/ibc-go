package testsuite

import (
	"testing"

	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/relayer"
	"go.uber.org/zap"

	"github.com/cosmos/ibc-go/e2e/testconfig"
)

const (
	cosmosRelayerRepository = "ghcr.io/cosmos/relayer"
)

// newCosmosRelayer returns an instance of the go relayer.
func newCosmosRelayer(t *testing.T, tc testconfig.TestConfig, logger *zap.Logger, dockerClient *dockerclient.Client, network string) ibc.Relayer {
	return ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger, relayer.CustomDockerImage(cosmosRelayerRepository, tc.RlyTag)).Build(
		t, dockerClient, network,
	)
}
