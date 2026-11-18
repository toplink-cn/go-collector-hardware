package bin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCommandHelperProcess(t *testing.T) {
	if os.Getenv("COLLECTOR_COMMAND_HELPER") != "1" {
		return
	}
	mode := os.Args[len(os.Args)-1]
	switch mode {
	case "sleep":
		time.Sleep(5 * time.Second)
	case "smart-failure":
		fmt.Print(`{"smart_status":{"passed":false}}`)
		fmt.Fprint(os.Stderr, "device reports failure")
		os.Exit(8)
	case "overflow":
		fmt.Print(strings.Repeat("x", MaxCommandOutput+1))
	}
	os.Exit(0)
}

func helper(t *testing.T, ctx context.Context, mode string) ([]byte, error) {
	t.Helper()
	t.Setenv("COLLECTOR_COMMAND_HELPER", "1")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return runCommand(ctx, executable, "-test.run=TestCommandHelperProcess", "--", mode)
}

func TestCommandHonorsParentDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := helper(t, ctx, "sleep")
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 2*time.Second {
		t.Fatalf("deadline not honored: %v", err)
	}
}
func TestNonzeroExitPreservesJSONStdoutSeparately(t *testing.T) {
	output, err := helper(t, context.Background(), "smart-failure")
	code, ok := ExitCode(err)
	if !ok || code != 8 || string(output) != `{"smart_status":{"passed":false}}` {
		t.Fatalf("output=%s error=%v code=%d", output, err, code)
	}
}
func TestCommandOutputIsBounded(t *testing.T) {
	output, err := helper(t, context.Background(), "overflow")
	if err == nil || len(output) != MaxCommandOutput {
		t.Fatalf("output length=%d error=%v", len(output), err)
	}
}
