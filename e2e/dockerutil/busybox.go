package dockerutil

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

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
