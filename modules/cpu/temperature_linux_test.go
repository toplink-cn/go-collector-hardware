//go:build linux

package cpu

import (
	"collector/modules/collection"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestMissingTemperatureSensorsAreUnsupportedWithoutLosingCPU(t *testing.T) {
	points, err := temperaturesFrom(context.Background(), t.TempDir())
	if len(points) != 0 || !errors.Is(err, collection.ErrUnsupported) {
		t.Fatalf("points=%v err=%v", points, err)
	}
}

func TestTemperatureScannerDoesNotRequireThermalZoneZero(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "class/hwmon/hwmon7")
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"name": "coretemp\n", "temp1_input": "45000\n", "temp1_label": "Package id 0\n", "temp2_input": "NaN\n"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	points, err := temperaturesFrom(context.Background(), root)
	if len(points) != 1 || points[0].Value != "45.00" || points[0].Label != "coretemp Package id 0" || err == nil {
		t.Fatalf("points=%+v err=%v", points, err)
	}
}
