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
	dockerclient "github.com/docker/docker/client"

	"github.com/cosmos/ibc-go/e2e/dockerutil"
)

const (
	dockerInspectFileName = "docker-inspect.json"
	e2eDir                = "e2e"
	defaultFilePerm       = 0o750
)

// Collect can be used in `t.Cleanup` and will copy all the of the container logs and relevant files
// into e2e/<test-suite>/<test-name>.log. These log files will be uploaded to GH upon test failure.
func Collect(t *testing.T, dc *dockerclient.Client, debugModeEnabled bool, chainNames ...string) {
	t.Helper()

	if !debugModeEnabled {
		// when we are not forcing log collection, we only upload upon test failing.
		if !t.Failed() {
			t.Logf("test passed, not uploading logs")
			return
		}
	}

	t.Logf("writing logs for test: %s", t.Name())

	ctx := context.TODO()
	e2eDir, err := getE2EDir(t)
	if err != nil {
		t.Logf("failed finding log directory: %s", err)
		return
	}

	logsDir := fmt.Sprintf("%s/diagnostics", e2eDir)

	if err := os.MkdirAll(fmt.Sprintf("%s/%s", logsDir, t.Name()), defaultFilePerm); err != nil {
		t.Logf("failed creating logs directory in test cleanup: %s", err)
		return
	}

	testContainers, err := dockerutil.GetTestContainers(ctx, t, dc)
	if err != nil {
		t.Logf("failed listing containers test cleanup: %s", err)
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
			t.Logf("failed writing log file for container %s in test cleanup: %s", containerName, err)
			continue
		}

		t.Logf("successfully wrote log file %s", logFile)

		var diagnosticFiles []string
		for _, chainName := range chainNames {
			diagnosticFiles = append(diagnosticFiles, chainDiagnosticAbsoluteFilePaths(chainName)...)
		}

		for _, absoluteFilePathInContainer := range diagnosticFiles {
			localFilePath := ospath.Join(containerDir, ospath.Base(absoluteFilePathInContainer))
			if err := fetchAndWriteDiagnosticsFile(ctx, dc, container.ID, localFilePath, absoluteFilePathInContainer); err != nil {
				t.Logf("failed to fetch and write file %s for container %s in test cleanup: %s", absoluteFilePathInContainer, containerName, err)
				continue
			}
			t.Logf("successfully wrote diagnostics file %s", absoluteFilePathInContainer)
		}

		localFilePath := ospath.Join(containerDir, dockerInspectFileName)
		if err := fetchAndWriteDockerInspectOutput(ctx, dc, container.ID, localFilePath); err != nil {
			t.Logf("failed to fetch docker inspect output: %s", err)
			continue
		}
		t.Logf("successfully wrote docker inspect output")
	}
}

// getContainerName returns a either the ID of the container or stripped down human-readable
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

// getE2EDir finds the e2e directory above the test.
func getE2EDir(t *testing.T) (string, error) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	const maxAttempts = 100
	count := 0
	for ; !strings.HasSuffix(wd, e2eDir) || count > maxAttempts; wd = ospath.Dir(wd) {
		count++
	}

	// arbitrary value to avoid getting stuck in an infinite loop if this is called
	// in a context where the e2e directory does not exist.
	if count > maxAttempts {
		return "", fmt.Errorf("unable to find e2e directory after %d tries", maxAttempts)
	}

	return wd, nil
}
