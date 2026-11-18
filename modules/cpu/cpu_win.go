//go:build windows

package cpu

import (
	"collector/bin"
	"collector/modules/collection"
	"context"
	"errors"
	"fmt"
	"os"
)

func collectTemperature(ctx context.Context) ([]CpuAttr, error) {
	output, err := bin.RunCommandContext(ctx, "OpenHardwareMonitorReport.exe")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("CPU temperature tool: %w", collection.ErrUnsupported)
		}
		return nil, err
	}
	values := parseWindowsTemperatures(string(output))
	if len(values) == 0 {
		return values, collection.ErrUnsupported
	}
	return values, nil
}
