package rollkit

// NOTE: this file should be able to be removed when https://github.com/strangelove-ventures/interchaintest/issues/869 is resolved

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/cosmos/ibc-go/e2e/dockerutil"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
	"io"
	"path"
	"sync"
	"time"
)

// FileRetriever allows retrieving a single file from a Docker volume.
// In the future it may allow retrieving an entire directory.
type FileRetriever struct {
	log *zap.Logger

	cli *client.Client

	testName string
}

// NewFileRetriever returns a new FileRetriever.
func NewFileRetriever(log *zap.Logger, cli *client.Client, testName string) *FileRetriever {
	return &FileRetriever{log: log, cli: cli, testName: testName}
}

// SingleFileContent returns the content of the file named at relPath,
// inside the volume specified by volumeName.
func (r *FileRetriever) SingleFileContent(ctx context.Context, volumeName, relPath string) ([]byte, error) {
	const mountPath = "/mnt/dockervolume"

	if err := ensureBusybox(ctx, r.cli); err != nil {
		return nil, err
	}

	containerName := fmt.Sprintf("interchaintest-getfile-%d-%s", time.Now().UnixNano(), dockerutil.RandLowerCaseLetterString(5))

	cc, err := r.cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: busyboxRef,

			// Use root user to avoid permission issues when reading files from the volume.
			User: dockerutil.GetRootUserString(),

			Labels: map[string]string{dockerutil.CleanupLabel: r.testName},
		},
		&container.HostConfig{
			Binds:      []string{volumeName + ":" + mountPath},
			AutoRemove: true,
		},
		nil, // No networking necessary.
		nil,
		containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("creating container: %w", err)
	}

	defer func() {
		if err := r.cli.ContainerRemove(ctx, cc.ID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			r.log.Warn("Failed to remove file content container", zap.String("container_id", cc.ID), zap.Error(err))
		}
	}()

	rc, _, err := r.cli.CopyFromContainer(ctx, cc.ID, path.Join(mountPath, relPath))
	if err != nil {
		return nil, fmt.Errorf("copying from container: %w", err)
	}
	defer func() {
		_ = rc.Close()
	}()

	wantPath := path.Base(relPath)
	tr := tar.NewReader(rc)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar from container: %w", err)
		}
		if hdr.Name != wantPath {
			r.log.Debug("Unexpected path", zap.String("want", relPath), zap.String("got", hdr.Name))
			continue
		}

		return io.ReadAll(tr)
	}

	return nil, fmt.Errorf("path %q not found in tar from container", relPath)
}

// Allow multiple goroutines to check for busybox
// by using a protected package-level variable.
//
// A mutex allows for retries upon error, if we ever need that;
// whereas a sync.Once would not be simple to retry.
var (
	ensureBusyboxMu sync.Mutex
	hasBusybox      bool
)

const busyboxRef = "busybox:stable"

func ensureBusybox(ctx context.Context, cli *client.Client) error {
	ensureBusyboxMu.Lock()
	defer ensureBusyboxMu.Unlock()

	if hasBusybox {
		return nil
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", busyboxRef)),
	})
	if err != nil {
		return fmt.Errorf("listing images to check busybox presence: %w", err)
	}

	if len(images) > 0 {
		hasBusybox = true
		return nil
	}

	rc, err := cli.ImagePull(ctx, busyboxRef, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()

	hasBusybox = true
	return nil
}
