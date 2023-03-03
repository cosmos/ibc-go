package dockerutil

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
)

const testLabel = "ibc-test"

// GetTestContainers returns all docker containers that have been created by interchain test.
func GetTestContainers(t *testing.T, ctx context.Context, dc *dockerclient.Client) ([]dockertypes.Container, error) {
	t.Helper()

	testContainers, err := dc.ContainerList(ctx, dockertypes.ContainerListOptions{
		All: true,
		Filters: filters.NewArgs(
			// see: https://github.com/strangelove-ventures/interchaintest/blob/0bdc194c2aa11aa32479f32b19e1c50304301981/internal/dockerutil/setup.go#L31-L36
			// for the label needed to identify test containers.
			filters.Arg("label", testLabel+"="+t.Name()),
		),
	})

	if err != nil {
		return nil, fmt.Errorf("failed listing containers: %s", err)
	}

	return testContainers, nil
}

func GetContainerLogs(ctx context.Context, dc *dockerclient.Client, containerName string) ([]byte, error) {
	readCloser, err := dc.ContainerLogs(ctx, containerName, dockertypes.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed reading logs in test cleanup: %s", err)
	}

	b := new(bytes.Buffer)
	_, err = b.ReadFrom(readCloser)
	if err != nil {
		return nil, fmt.Errorf("failed reading logs in test cleanup: %s", err)
	}

	return b.Bytes(), nil
}
