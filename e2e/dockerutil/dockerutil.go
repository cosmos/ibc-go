package dockerutil

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"path"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/moby/moby/client"
)

const testLabel = "ibc-test"

// GetTestContainers returns all docker containers that have been created by interchain test.
// note: the test suite name must be passed as the chains are created in SetupSuite which will label
// them with the name of the test suite rather than the test.
func GetTestContainers(ctx context.Context, suiteName string, dc *dockerclient.Client) ([]dockertypes.Container, error) {
	testContainers, err := dc.ContainerList(ctx, container.ListOptions{
		All: true,
		Filters: filters.NewArgs(
			// see: https://github.com/cosmos/interchaintest/blob/0bdc194c2aa11aa32479f32b19e1c50304301981/internal/dockerutil/setup.go#L31-L36
			// for the suiteName needed to identify test containers.
			filters.Arg("label", testLabel+"="+suiteName),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed listing containers: %w", err)
	}

	return testContainers, nil
}

// GetContainerLogs returns the logs of a container as a byte array.
func GetContainerLogs(ctx context.Context, dc *dockerclient.Client, containerName string) ([]byte, error) {
	readCloser, err := dc.ContainerLogs(ctx, containerName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed reading logs in test cleanup: %w", err)
	}
	return io.ReadAll(readCloser)
}

// GetFileContentsFromContainer reads the contents of a specific file from a container.
func GetFileContentsFromContainer(ctx context.Context, dc *dockerclient.Client, containerID, absolutePath string) ([]byte, error) {
	readCloser, _, err := dc.CopyFromContainer(ctx, containerID, absolutePath)
	if err != nil {
		return nil, fmt.Errorf("copying from container: %w", err)
	}

	defer readCloser.Close()

	fileName := path.Base(absolutePath)
	tr := tar.NewReader(readCloser)

	hdr, err := tr.Next()
	if err != nil {
		return nil, err
	}

	if hdr.Name != fileName {
		return nil, fmt.Errorf("expected to find %s but found %s", fileName, hdr.Name)
	}

	return io.ReadAll(tr)
}
