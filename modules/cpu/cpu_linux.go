//go:build linux

package cpu

import (
	"collector/modules/collection"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func collectTemperature(ctx context.Context) ([]CpuAttr, error) { return temperaturesFrom(ctx, "/sys") }

func temperaturesFrom(ctx context.Context, root string) ([]CpuAttr, error) {
	files, _ := filepath.Glob(filepath.Join(root, "class/hwmon/hwmon*/temp*_input"))
	sort.Strings(files)
	values := []CpuAttr{}
	problems := []error{}
	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return values, errors.Join(append(problems, err)...)
		}
		nameBytes, err := os.ReadFile(filepath.Join(filepath.Dir(path), "name"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(nameBytes))
		if name != "coretemp" && name != "k10temp" && name != "zenpower" && name != "cpu_thermal" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
		value /= 1000
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > 200 {
			problems = append(problems, fmt.Errorf("invalid CPU temperature %s", filepath.Base(path)))
			continue
		}
		labelBytes, _ := os.ReadFile(strings.TrimSuffix(path, "_input") + "_label")
		label := strings.TrimSpace(string(labelBytes))
		if label == "" {
			label = filepath.Base(path)
		}
		source, _ := filepath.EvalSymlinks(path)
		if source == "" {
			source = path
		}
		source = strings.TrimPrefix(source, root)
		source = regexp.MustCompile(`/hwmon/hwmon[0-9]+/`).ReplaceAllString(source, "/hwmon/")
		values = append(values, CpuAttr{ID: sensorID(source), Value: strconv.FormatFloat(value, 'f', 2, 64), Label: name + " " + label})
	}
	if len(values) == 0 && len(problems) == 0 {
		return values, collection.ErrUnsupported
	}
	return values, errors.Join(problems...)
}
