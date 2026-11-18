//go:build linux

package ipmi

import (
	"collector/modules/collection"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestIPMIParserSkipsUnavailableSensorsAndRetainsUnits(t *testing.T) {
	values := parseSensors("CPU Temp | 42.0 | degrees C | ok\nAbsent | na | unspecified | na\nFan | 0 | RPM | ok\n")
	if len(values) != 2 || values[0].Value != "42.0 degrees C" || values[1].Value != "0 RPM" {
		t.Fatalf("%+v", values)
	}
}

func TestMissingLocalIPMIIsUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ipmi0")
	if err := checkLocalDevice([]string{path}); !errors.Is(err, collection.ErrUnsupported) {
		t.Fatalf("%v", err)
	}
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkLocalDevice([]string{path}); err != nil {
		t.Fatal(err)
	}
}
