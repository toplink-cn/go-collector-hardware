package cpu

import (
	"collector/modules/collection"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	gocpu "github.com/shirou/gopsutil/v3/cpu"
)

type CpuObj struct {
	Usage       []CpuAttr
	Temperature []CpuAttr
}
type CpuAttr struct {
	ID    string
	Value string
	Label string `json:"Label,omitempty"`
}

func Collect(ctx context.Context) (collection.Result[CpuObj], error) {
	return collectCPU(ctx, func(ctx context.Context) ([]CpuAttr, error) {
		return collectUsage(ctx, gocpu.TimesWithContext, time.Second)
	}, collectTemperature, time.Now)
}

type metricReader func(context.Context) ([]CpuAttr, error)

func collectCPU(ctx context.Context, readUsage, readTemperature metricReader, now func() time.Time) (collection.Result[CpuObj], error) {
	usage, usageErr := readUsage(ctx)
	usageAt := now().Unix()
	temperatures, temperatureErr := readTemperature(ctx)
	temperatureAt := now().Unix()
	cpuQuality := collection.FromError(usageErr, len(usage) > 0)
	temperatureQuality := collection.FromError(temperatureErr, len(temperatures) > 0)
	if len(usage) > 0 && (cpuQuality.State == collection.Success || cpuQuality.State == collection.Partial) {
		cpuQuality.CollectedAt = usageAt
	}
	if len(temperatures) > 0 && (temperatureQuality.State == collection.Success || temperatureQuality.State == collection.Partial) {
		temperatureQuality.CollectedAt = temperatureAt
	}
	return collection.Result[CpuObj]{Data: CpuObj{Usage: usage, Temperature: temperatures}, Quality: map[string]collection.Quality{
		"cpu":         cpuQuality,
		"temperature": temperatureQuality,
	}}, errors.Join(usageErr, temperatureErr)
}

func GetInfo() CpuObj { result, _ := Collect(context.Background()); return result.Data }

type timesReader func(context.Context, bool) ([]gocpu.TimesStat, error)

func collectUsage(ctx context.Context, read timesReader, interval time.Duration) ([]CpuAttr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	first, err := read(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("CPU first sample: %w", err)
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}
	second, err := read(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("CPU second sample: %w", err)
	}
	return usageDelta(first, second)
}

func usageDelta(first, second []gocpu.TimesStat) ([]CpuAttr, error) {
	before := map[string]gocpu.TimesStat{}
	for _, point := range first {
		before[point.CPU] = point
	}
	values := []CpuAttr{}
	problems := []error{}
	seen := map[string]bool{}
	for _, point := range second {
		id := strings.TrimPrefix(point.CPU, "cpu")
		if _, err := strconv.ParseUint(id, 10, 32); err != nil || seen[id] {
			problems = append(problems, fmt.Errorf("invalid or duplicate logical CPU %q", point.CPU))
			continue
		}
		seen[id] = true
		old, ok := before[point.CPU]
		if !ok {
			problems = append(problems, fmt.Errorf("CPU %s appeared between samples", id))
			continue
		}
		// Guest counters are already included in user/nice on Linux.
		current := []float64{point.User, point.Nice, point.System, point.Idle, point.Iowait, point.Irq, point.Softirq, point.Steal}
		previous := []float64{old.User, old.Nice, old.System, old.Idle, old.Iowait, old.Irq, old.Softirq, old.Steal}
		total, idle := float64(0), float64(0)
		valid := true
		for index, value := range current {
			delta := value - previous[index]
			if math.IsNaN(delta) || math.IsInf(delta, 0) || delta < 0 {
				valid = false
				break
			}
			total += delta
			if index == 3 || index == 4 {
				idle += delta
			}
		}
		if !valid || total <= 0 || math.IsInf(total, 0) {
			problems = append(problems, fmt.Errorf("CPU %s counter reset or missing interval", id))
			continue
		}
		usage := (total - idle) / total * 100
		values = append(values, CpuAttr{ID: id, Value: strconv.FormatFloat(usage, 'f', 2, 64)})
	}
	if len(first) != len(second) {
		problems = append(problems, errors.New("CPU topology changed between samples"))
	}
	sort.Slice(values, func(i, j int) bool {
		a, _ := strconv.Atoi(values[i].ID)
		b, _ := strconv.Atoi(values[j].ID)
		return a < b
	})
	if len(values) == 0 {
		problems = append(problems, errors.New("no valid CPU interval samples"))
	}
	return values, errors.Join(problems...)
}

func sensorID(source string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(source))
	return strconv.FormatUint(uint64(hash.Sum32()), 10)
}
