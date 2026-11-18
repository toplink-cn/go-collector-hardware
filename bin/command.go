package bin

import (
	"bytes"
	"collector/utils"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

const CmdTimeout = 10 * time.Second
const MaxCommandOutput = 2 * 1024 * 1024

// RunCommandContext returns stdout separately so diagnostics cannot corrupt JSON.
// A shorter parent deadline always wins over the per-command limit.
func RunCommandContext(parent context.Context, filename string, args ...string) ([]byte, error) {
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(utils.GetBinDir(), filepath.FromSlash(filename))
	}
	return runCommand(parent, filename, args...)
}

func runCommand(parent context.Context, filename string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, CmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, filename, args...)
	cmd.WaitDelay = 250 * time.Millisecond
	stdout, stderr := &boundedBuffer{}, &boundedBuffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return stdout.Bytes(), fmt.Errorf("%s: %w", filepath.Base(filename), ctx.Err())
	}
	if stdout.exceeded || stderr.exceeded {
		return stdout.Bytes(), fmt.Errorf("%s: command output exceeded limit", filepath.Base(filename))
	}
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("%s: %w: %s", filepath.Base(filename), err, diagnostic(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func ExitCode(err error) (int, bool) {
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) {
		return exit.ExitCode(), true
	}
	return 0, false
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (b *boundedBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *boundedBuffer) String() string { return b.buffer.String() }
func diagnostic(message string) string {
	if len(message) > 4096 {
		return message[:4096] + " (truncated)"
	}
	return message
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	count := len(p)
	remaining := MaxCommandOutput - b.buffer.Len()
	if count > remaining {
		b.exceeded = true
		p = p[:remaining]
	}
	_, _ = b.buffer.Write(p)
	return count, nil
}

func RunCommand(filename string, args ...string) ([]byte, error) {
	return RunCommandContext(context.Background(), filename, args...)
}

// Deprecated: callers needing quality evidence must use RunCommandContext.
func RunCommandAndReturnBytes(filename string, args ...string) bytes.Buffer {
	out, _ := RunCommand(filename, args...)
	return *bytes.NewBuffer(out)
}
