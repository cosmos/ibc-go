package dockerutil

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

// FileWriter allows retrieving a single file from a Docker volume.
// In the future it may allow retrieving an entire directory.
type FileWriter struct {
	log *zap.Logger

	cli *client.Client

	testName string
}

// NewFileWriter returns a new FileWriter.
func NewFileWriter(log *zap.Logger, cli *client.Client, testName string) *FileWriter {
	return &FileWriter{log: log, cli: cli, testName: testName}
}

// WriteFile writes the single file containing content, at relPath within the given volume.
func (w *FileWriter) WriteFile(ctx context.Context, volumeName, relPath string, content []byte) error {
	const mountPath = "/mnt/dockervolume"

	if err := ensureBusybox(ctx, w.cli); err != nil {
		return err
	}

	containerName := fmt.Sprintf("interchaintest-writefile-%d-%s", time.Now().UnixNano(), RandLowerCaseLetterString(5))

	cc, err := w.cli.ContainerCreate(
		ctx,
		&container.Config{
			Image: busyboxRef,

			Entrypoint: []string{"sh", "-c"},
			Cmd: []string{
				// Take the uid and gid of the mount path,
				// and set that as the owner of the new relative path.
				`chown -R "$(stat -c '%u:%g' "$1")" "$2"`,
				"_", // Meaningless arg0 for sh -c with positional args.
				mountPath,
				mountPath,
			},

			// Use root user to avoid permission issues when reading files from the volume.
			User: GetRootUserString(),

			Labels: map[string]string{CleanupLabel: w.testName},
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
		return fmt.Errorf("creating container: %w", err)
	}

	autoRemoved := false
	defer func() {
		if autoRemoved {
			// No need to attempt removing the container if we successfully started and waited for it to complete.
			return
		}

		if err := w.cli.ContainerRemove(ctx, cc.ID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			w.log.Warn("Failed to remove file content container", zap.String("container_id", cc.ID), zap.Error(err))
		}
	}()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{
		Name: relPath,

		Size: int64(len(content)),
		Mode: 0600,
		// Not setting uname because the container will chown it anyway.

		ModTime: time.Now(),

		Format: tar.FormatPAX,
	}); err != nil {
		return fmt.Errorf("writing tar header: %w", err)
	}
	if _, err := tw.Write(content); err != nil {
		return fmt.Errorf("writing content to tar: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("closing tar writer: %w", err)
	}

	if err := w.cli.CopyToContainer(
		ctx,
		cc.ID,
		mountPath,
		&buf,
		types.CopyToContainerOptions{},
	); err != nil {
		return fmt.Errorf("copying tar to container: %w", err)
	}

	if err := w.cli.ContainerStart(ctx, cc.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("starting write-file container: %w", err)
	}

	waitCh, errCh := w.cli.ContainerWait(ctx, cc.ID, container.WaitConditionNotRunning)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	case res := <-waitCh:
		autoRemoved = true

		if res.Error != nil {
			return fmt.Errorf("waiting for write-file container: %s", res.Error.Message)
		}

		if res.StatusCode != 0 {
			return fmt.Errorf("chown on new file exited %d", res.StatusCode)
		}
	}

	return nil
}
