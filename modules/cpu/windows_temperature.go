package cpu

import (
	"regexp"
	"strconv"
	"strings"
)

func parseWindowsTemperatures(output string) []CpuAttr {
	expression := regexp.MustCompile(`CPU Core #(\d+)\s*:\s*([0-9]+(?:\.[0-9]+)?).*\(([^)]*/temperature/[^)]*)\)`)
	values := []CpuAttr{}
	for _, line := range strings.Split(output, "\n") {
		match := expression.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}
		value, err := strconv.ParseFloat(match[2], 64)
		if err != nil || value <= 0 || value > 200 {
			continue
		}
		values = append(values, CpuAttr{ID: sensorID(match[3]), Value: strconv.FormatFloat(value, 'f', 2, 64), Label: "CPU Core #" + match[1]})
	}
	return values
}
