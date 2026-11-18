//go:build linux

package ipmi

import (
	"collector/modules/collection"
	"os"
	"strings"
)

func platformAvailability() error {
	return checkLocalDevice([]string{"/dev/ipmi0", "/dev/ipmi/0", "/dev/ipmidev/0"})
}
func checkLocalDevice(paths []string) error {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
	}
	return collection.ErrUnsupported
}

func ipmiExecutable() string { return "ipmitool" }
func parseSensors(output string) []Sensor {
	values := []Sensor{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "|")
		if len(fields) < 3 {
			continue
		}
		name, value, unit := strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1]), strings.TrimSpace(fields[2])
		if name == "" || value == "" || strings.EqualFold(value, "na") || strings.EqualFold(value, "no reading") {
			continue
		}
		if unit != "" && !strings.EqualFold(unit, "unspecified") {
			value += " " + unit
		}
		values = append(values, Sensor{Key: name, Value: value})
	}
	return values
}
