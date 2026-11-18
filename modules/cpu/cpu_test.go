package cpu

import (
	"collector/modules/collection"
	"context"
	"errors"
	"math"
	"testing"
	"time"

	gocpu "github.com/shirou/gopsutil/v3/cpu"
)

func TestUsageIsIntervalDeltaAndDoesNotMergeSiblingThreads(t *testing.T) {
	first := []gocpu.TimesStat{{CPU: "cpu0", User: 1000, Idle: 1000, Guest: 100}, {CPU: "cpu1", User: 1000, Idle: 1000}}
	second := []gocpu.TimesStat{{CPU: "cpu1", User: 1010, Idle: 1090}, {CPU: "cpu0", User: 1080, Idle: 1020, Guest: 180}}
	points, err := usageDelta(first, second)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 || points[0].ID != "0" || points[0].Value != "80.00" || points[1].ID != "1" || points[1].Value != "10.00" {
		t.Fatalf("unexpected logical CPU delta: %+v", points)
	}
}

func TestIowaitAndGuestAreNotCountedAsBusyTwice(t *testing.T) {
	first := []gocpu.TimesStat{{CPU: "cpu0"}}
	second := []gocpu.TimesStat{{CPU: "cpu0", User: 20, Idle: 30, Iowait: 50, Guest: 20}}
	points, err := usageDelta(first, second)
	if err != nil || len(points) != 1 || points[0].Value != "20.00" {
		t.Fatalf("points=%+v error=%v", points, err)
	}
}

func TestResetsAndMissingIntervalsKeepValidCoreAsPartial(t *testing.T) {
	first := []gocpu.TimesStat{{CPU: "cpu0", User: 10, Idle: 10}, {CPU: "cpu1", User: 10, Idle: 10}}
	second := []gocpu.TimesStat{{CPU: "cpu0", User: 20, Idle: 20}, {CPU: "cpu1", User: 1, Idle: 1}}
	points, err := usageDelta(first, second)
	if len(points) != 1 || err == nil {
		t.Fatalf("points=%+v error=%v", points, err)
	}
	_, err = usageDelta(first, first)
	if err == nil {
		t.Fatal("zero interval must be unavailable")
	}
	second[0].User = math.NaN()
	points, err = usageDelta(first, second)
	if len(points) != 0 || err == nil {
		t.Fatal("NaN and resets must never become measurements")
	}
}

func TestCancellationInterruptsSamplingWithoutTakingASecondReading(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	read := func(context.Context, bool) ([]gocpu.TimesStat, error) {
		calls++
		cancel()
		return []gocpu.TimesStat{{CPU: "cpu0"}}, nil
	}
	started := time.Now()
	_, err := collectUsage(ctx, read, time.Hour)
	if !errors.Is(err, context.Canceled) || calls != 1 || time.Since(started) > time.Second {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestWindowsTemperatureParserPreservesStableSensorIdentity(t *testing.T) {
	output := "CPU Core #1 : 45.5 30 60 (/intelcpu/0/temperature/0)\nCPU Core #2 : 0 0 0 (/intelcpu/0/temperature/1)\n"
	points := parseWindowsTemperatures(output)
	if len(points) != 1 || points[0].Value != "45.50" || points[0].Label != "CPU Core #1" {
		t.Fatalf("%+v", points)
	}
	if sensorID("/intelcpu/0/temperature/0") != points[0].ID {
		t.Fatal("unstable sensor identity")
	}
}

func TestUsageTimeIsCapturedBeforeSlowOptionalTemperature(t *testing.T) {
	moments := []int64{1700000001, 1700000011}
	index := 0
	now := func() time.Time { moment := time.Unix(moments[index], 0); index++; return moment }
	usage := func(context.Context) ([]CpuAttr, error) { return []CpuAttr{{ID: "0", Value: "25.00"}}, nil }
	temperature := func(context.Context) ([]CpuAttr, error) {
		return []CpuAttr{{ID: "1", Value: "45.00"}}, errors.New("another sensor unreadable")
	}
	result, err := collectCPU(context.Background(), usage, temperature, now)
	if err == nil || result.Quality["cpu"].CollectedAt != 1700000001 || result.Quality["temperature"].CollectedAt != 1700000011 || result.Quality["temperature"].State != collection.Partial {
		t.Fatalf("lost independent observation times: %+v error=%v", result.Quality, err)
	}
}

func TestFailedOrUnsupportedGroupsHaveNoInventedObservationTime(t *testing.T) {
	usage := func(context.Context) ([]CpuAttr, error) { return nil, context.DeadlineExceeded }
	temperature := func(context.Context) ([]CpuAttr, error) { return nil, collection.ErrUnsupported }
	result, err := collectCPU(context.Background(), usage, temperature, func() time.Time { return time.Unix(1700000011, 0) })
	if err == nil || result.Quality["cpu"].CollectedAt != 0 || result.Quality["temperature"].CollectedAt != 0 {
		t.Fatalf("invented observation time: %+v", result.Quality)
	}
}
