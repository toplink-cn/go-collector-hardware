//go:build windows

package ipmi

import (
	"regexp"
	"strings"
)

func platformAvailability() error { return nil }

func ipmiExecutable() string { return "ipmiutil/ipmiutil.exe" }
func parseSensors(output string) []Sensor {
	values := []Sensor{}
	pattern := regexp.MustCompile(`[0-9A-Fa-f]+\s+.*\s+snum [0-9A-Fa-f]+ (.*)\s+\= [0-9A-Fa-f]+ (?:OK)?(.*)?`)
	for _, line := range strings.Split(output, "\n") {
		match := pattern.FindStringSubmatch(line)
		if len(match) < 3 {
			continue
		}
		name, value := strings.TrimSpace(match[1]), strings.TrimSpace(match[2])
		if name != "" && value != "" && !strings.EqualFold(value, "na") {
			values = append(values, Sensor{Key: name, Value: value})
		}
	}
	return values
}
