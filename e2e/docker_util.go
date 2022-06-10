package e2e

import (
	"bytes"
	"context"
	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func DockerExec(t *testing.T, ctx context.Context, cli dockerClient.APIClient, id string, cmd []string) ExecResult {
	res, err := Exec(ctx, cli, id, cmd, true)
	assert.NoError(t, err)
	t.Logf("Command: %s", strings.Join(cmd, " "))
	t.Logf("Output: %+v", res.Combined())
	assert.Equal(t, res.ExitCode, 0)
	return res
}

func DockerExecUnattached(t *testing.T, ctx context.Context, cli dockerClient.APIClient, id string, cmd []string) ExecResult {
	res, err := Exec(ctx, cli, id, cmd, false)
	assert.NoError(t, err)
	t.Logf("Command: %s", strings.Join(cmd, " "))
	t.Logf("Output: %+v", res.Combined())
	assert.Equal(t, res.ExitCode, 0)
	return res
}

// Exec executes a command in the given docker container. This function is necessary to strip  leading bytes
// which denote stdin/stderr. This was originally found on this Stack Overflow question
// https://stackoverflow.com/questions/52774830/docker-exec-command-from-golang-api
func Exec(ctx context.Context, cli dockerClient.APIClient, id string, cmd []string, attached bool) (ExecResult, error) {
	// prepare exec
	execConfig := dockerTypes.ExecConfig{
		AttachStdout: attached,
		AttachStderr: attached,
		Cmd:          cmd,
	}
	cresp, err := cli.ContainerExecCreate(ctx, id, execConfig)
	if err != nil {
		return ExecResult{}, err
	}
	execID := cresp.ID

	// run it, with stdout/stderr attached
	aresp, err := cli.ContainerExecAttach(ctx, execID, dockerTypes.ExecStartCheck{})
	if err != nil {
		return ExecResult{}, err
	}
	defer aresp.Close()

	// read the output
	var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	go func() {
		// StdCopy demultiplexes the stream into two buffers
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, aresp.Reader)
		outputDone <- err
	}()

	select {
	case err := <-outputDone:
		if err != nil {
			return ExecResult{}, err
		}
		break

	case <-ctx.Done():
		return ExecResult{}, ctx.Err()
	}

	// get the exit code
	iresp, err := cli.ContainerExecInspect(ctx, execID)
	if err != nil {
		return ExecResult{}, err
	}

	return ExecResult{ExitCode: iresp.ExitCode, outBuffer: &outBuf, errBuffer: &errBuf}, nil
}

// ExecResult represents a result returned from Exec()
type ExecResult struct {
	ExitCode  int
	outBuffer *bytes.Buffer
	errBuffer *bytes.Buffer
}

// Stdout returns stdout output of a command run by Exec()
func (res *ExecResult) Stdout() string {
	return res.outBuffer.String()
}

// Stderr returns stderr output of a command run by Exec()
func (res *ExecResult) Stderr() string {
	return res.errBuffer.String()
}

// Combined returns combined stdout and stderr output of a command run by Exec()
func (res *ExecResult) Combined() string {
	return res.outBuffer.String() + res.errBuffer.String()
}
