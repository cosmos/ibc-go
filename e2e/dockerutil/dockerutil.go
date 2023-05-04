package dockerutil

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"path"
	"testing"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
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

// GetContainerLogs returns the logs of a container as a byte array.
func GetContainerLogs(ctx context.Context, dc *dockerclient.Client, containerName string) ([]byte, error) {
	readCloser, err := dc.ContainerLogs(ctx, containerName, dockertypes.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed reading logs in test cleanup: %s", err)
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

// SetGenesisContentsToContainer set the contents of a specific file to a container.
func SetGenesisContentsToContainer(t *testing.T, ctx context.Context, dc *dockerclient.Client, cfg ibc.ChainConfig, content io.Reader, options dockertypes.CopyToContainerOptions) error {
	containerID, err := getDockerContainerID(t, ctx, cfg, dc)
	if err != nil {
		return err
	}
	genesisFilePath := chainAbsoluteGenesisFilePaths(cfg)
	err = dc.CopyToContainer(ctx, containerID, genesisFilePath, content, options)
	if err != nil {
		return err
	}
	return nil
}

//chainAbsoluteGenesisFilePaths get absolute path of Genesis file of a chain
func chainAbsoluteGenesisFilePaths(cfg ibc.ChainConfig) string {
	return fmt.Sprintf("/var/cosmos-chain/%s/config/genesis.json", cfg.Name)
}

//getDockerContainerID get docker container id
func getDockerContainerID(t *testing.T, ctx context.Context, cfg ibc.ChainConfig, dc *dockerclient.Client) (string, error) {
	imageOfChain := cfg.Images[0].Repository
	testContainers, err := GetTestContainers(t, ctx, dc)
	if err != nil {
		return  "", err
	}
	for _, container := range testContainers {
		if container.Image == imageOfChain {
			return container.ID, nil
		}
	}
	return "", fmt.Errorf("can't find container id")
}