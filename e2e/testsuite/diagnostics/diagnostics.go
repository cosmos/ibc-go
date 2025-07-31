package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	ospath "path"
	"strings"
	"testing"

	dockertypes "github.com/docker/docker/api/types"
	dockerclient "github.com/moby/moby/client"

	"github.com/cosmos/ibc-go/e2e/dockerutil"
	"github.com/cosmos/ibc-go/e2e/internal/directories"
)

const (
	dockerInspectFileName = "docker-inspect.json"
	defaultFilePerm       = 0o750
)

// Collect can be used in `t.Cleanup` and will copy all the of the container logs and relevant files
// into e2e/<test-suite>/<test-name>.log. These log files will be uploaded to GH upon test failure.
func Collect(t *testing.T, dc *dockerclient.Client, debugModeEnabled bool, suiteName string, chainNames ...string) {
	t.Helper()

	if !debugModeEnabled {
		// when we are not forcing log collection, we only upload upon test failing.
		if !t.Failed() {
			t.Logf("test passed, not uploading logs")
			return
		}
	}

	t.Logf("writing logs for test: %s", t.Name())

	ctx := t.Context()
	e2eDir, err := directories.E2E()
	if err != nil {
		t.Logf("failed finding log directory: %s", err)
		return
	}

	logsDir := fmt.Sprintf("%s/diagnostics", e2eDir)

	if err := os.MkdirAll(fmt.Sprintf("%s/%s", logsDir, t.Name()), defaultFilePerm); err != nil {
		t.Logf("failed creating logs directory in test cleanup: %s", err)
		return
	}

	testContainers, err := dockerutil.GetTestContainers(ctx, suiteName, dc)
	if err != nil {
		t.Logf("failed listing containers during test cleanup: %s", err)
		return
	}

	for _, container := range testContainers {
		containerName := getContainerName(t, container)
		containerDir := fmt.Sprintf("%s/%s/%s", logsDir, t.Name(), containerName)
		if err := os.MkdirAll(containerDir, defaultFilePerm); err != nil {
			t.Logf("failed creating logs directory for container %s: %s", containerDir, err)
			continue
		}

		logsBz, err := dockerutil.GetContainerLogs(ctx, dc, container.ID)
		if err != nil {
			t.Logf("failed reading logs in test cleanup: %s", err)
			continue
		}

		logFile := fmt.Sprintf("%s/%s.log", containerDir, containerName)
		if err := os.WriteFile(logFile, logsBz, defaultFilePerm); err != nil {
			continue
		}

		t.Logf("successfully wrote log file %s", logFile)

		var diagnosticFiles []string
		for _, chainName := range chainNames {
			diagnosticFiles = append(diagnosticFiles, chainDiagnosticAbsoluteFilePaths(chainName)...)
		}
		diagnosticFiles = append(diagnosticFiles, relayerDiagnosticAbsoluteFilePaths()...)

		for _, absoluteFilePathInContainer := range diagnosticFiles {
			localFilePath := ospath.Join(containerDir, ospath.Base(absoluteFilePathInContainer))
			if err := fetchAndWriteDiagnosticsFile(ctx, dc, container.ID, localFilePath, absoluteFilePathInContainer); err != nil {
				continue
			}
			t.Logf("successfully wrote diagnostics file %s", absoluteFilePathInContainer)
		}

		localFilePath := ospath.Join(containerDir, dockerInspectFileName)
		if err := fetchAndWriteDockerInspectOutput(ctx, dc, container.ID, localFilePath); err != nil {
			continue
		}
		t.Logf("successfully wrote docker inspect output")
	}
}

// getContainerName returns an either the ID of the container or stripped down human-readable
// version of the name if the name is non-empty.
//
// Note: You should still always use the ID  when interacting with the docker client.
func getContainerName(t *testing.T, container dockertypes.Container) string {
	t.Helper()
	// container will always have an id, by may not have a name.
	containerName := container.ID
	if len(container.Names) > 0 {
		containerName = container.Names[0]
		// remove the test name from the container as the folder structure will provide this
		// information already.
		containerName = strings.TrimRight(containerName, "-"+t.Name())
		containerName = strings.TrimLeft(containerName, "/")
	}
	return containerName
}

// fetchAndWriteDiagnosticsFile fetches the contents of a single file from the given container id and writes
// the contents of the file to a local path provided.
func fetchAndWriteDiagnosticsFile(ctx context.Context, dc *dockerclient.Client, containerID, localPath, absoluteFilePathInContainer string) error {
	fileBz, err := dockerutil.GetFileContentsFromContainer(ctx, dc, containerID, absoluteFilePathInContainer)
	if err != nil {
		return err
	}

	return os.WriteFile(localPath, fileBz, defaultFilePerm)
}

// fetchAndWriteDockerInspectOutput writes the contents of docker inspect to the specified file.
func fetchAndWriteDockerInspectOutput(ctx context.Context, dc *dockerclient.Client, containerID, localPath string) error {
	containerJSON, err := dc.ContainerInspect(ctx, containerID)
	if err != nil {
		return err
	}

	fileBz, err := json.MarshalIndent(containerJSON, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(localPath, fileBz, defaultFilePerm)
}

// chainDiagnosticAbsoluteFilePaths returns a slice of absolute file paths (in the containers) which are the files that should be
// copied locally when fetching diagnostics.
func chainDiagnosticAbsoluteFilePaths(chainName string) []string {
	return []string{
		fmt.Sprintf("/var/cosmos-chain/%s/config/genesis.json", chainName),
		fmt.Sprintf("/var/cosmos-chain/%s/config/app.toml", chainName),
		fmt.Sprintf("/var/cosmos-chain/%s/config/config.toml", chainName),
		fmt.Sprintf("/var/cosmos-chain/%s/config/client.toml", chainName),
	}
}

// relayerDiagnosticAbsoluteFilePaths returns a slice of absolute file paths (in the containers) which are the files that should be
// copied locally when fetching diagnostics.
func relayerDiagnosticAbsoluteFilePaths() []string {
	return []string{
		"/home/hermes/.hermes/config.toml",
	}
}
