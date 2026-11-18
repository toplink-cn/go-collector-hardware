package disk

import (
	"collector/modules/collection"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

type exitStatus int

func (e exitStatus) Error() string { return fmt.Sprintf("exit %d", int(e)) }
func (e exitStatus) ExitCode() int { return int(e) }

func TestSMARTFailureBitsAreValidHealthObservations(t *testing.T) {
	runner := func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"model_name":"Disk","smart_status":{"passed":false},"user_capacity":{"bytes":1000},"temperature":{"current":45},"smartctl":{"exit_status":8}}`), exitStatus(8)
	}
	point, err := readDisk(context.Background(), runner, "smartctl", Device{Name: "/dev/sda"})
	if err != nil || point.SmartStatus.Passed == nil || *point.SmartStatus.Passed {
		t.Fatalf("point=%+v error=%v", point, err)
	}
}

func TestCommandFailureAndMissingSMARTNeverBecomeHealthyOrFailed(t *testing.T) {
	runner := func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"smart_status":{"passed":false},"temperature":{"current":0},"user_capacity":{"bytes":0}}`), exitStatus(2)
	}
	point, err := readDisk(context.Background(), runner, "smartctl", Device{Name: "/dev/sda"})
	if err == nil || point.SmartStatus.Passed != nil || point.Temperature.Current != nil || point.UserCapacity.Bytes != nil {
		t.Fatalf("point=%+v error=%v", point, err)
	}
	encoded, _ := json.Marshal(point)
	var raw map[string]interface{}
	_ = json.Unmarshal(encoded, &raw)
	if raw["smart_status"].(map[string]interface{})["passed"] != nil {
		t.Fatal("unknown SMART must encode as null")
	}
}

func TestDiskWorkersAreBoundedAndFailedDeviceCannotDeadlockOtherDevices(t *testing.T) {
	devices := []Device{}
	for index := 0; index < 12; index++ {
		devices = append(devices, Device{Name: fmt.Sprintf("/dev/disk%d", index), Type: "sat"})
	}
	scan, _ := json.Marshal(Smartctl{Devices: devices})
	var active, maxActive, completed atomic.Int32
	runner := func(ctx context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) == 2 {
			return scan, nil
		}
		current := active.Add(1)
		defer active.Add(-1)
		for {
			old := maxActive.Load()
			if current <= old || maxActive.CompareAndSwap(old, current) {
				break
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Millisecond):
		}
		completed.Add(1)
		if args[2] == "/dev/disk0" {
			return []byte(`invalid JSON`), errors.New("device failed")
		}
		return []byte(`{"model_name":"Disk","smart_status":{"passed":true},"user_capacity":{"bytes":1000}}`), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	values, err := collectDisks(ctx, runner, "smartctl", 3)
	if err == nil || len(values) != 12 || completed.Load() != 12 || maxActive.Load() > 3 {
		t.Fatalf("values=%d complete=%d active=%d error=%v", len(values), completed.Load(), maxActive.Load(), err)
	}
	if collection.FromError(err, len(values) > 0).State != collection.Partial {
		t.Fatal("valid devices must survive partial failure")
	}
}

func TestCanceledDiskCollectionStopsAllWorkers(t *testing.T) {
	scan := []byte(`{"devices":[{"name":"/dev/a"},{"name":"/dev/b"},{"name":"/dev/c"}]}`)
	runner := func(ctx context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) == 2 {
			return scan, nil
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := collectDisks(ctx, runner, "smartctl", 2)
	if err == nil || time.Since(started) > time.Second {
		t.Fatalf("cancellation failed: %v", err)
	}
}

func TestNVMeCapacityAndUnknownRotationKeepTheirEvidence(t *testing.T) {
	runner := func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"device":{"protocol":"NVMe"},"smart_status":{"passed":true},"nvme_total_capacity":4096}`), nil
	}
	point, err := readDisk(context.Background(), runner, "smartctl", Device{Name: "/dev/nvme0n1"})
	if err != nil || point.UserCapacity.Bytes == nil || *point.UserCapacity.Bytes != 4096 || point.ModelType != "NVMe" || point.SerialNumber != "" {
		t.Fatalf("point=%+v error=%v", point, err)
	}
}

func TestRAIDMembersSharingDevicePathKeepDistinctIdentity(t *testing.T) {
	runner := func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"smart_status":{"passed":true},"user_capacity":{"bytes":1000}}`), nil
	}
	first, err := readDisk(context.Background(), runner, "smartctl", Device{Name: "/dev/bus/0", Type: "megaraid,0"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := readDisk(context.Background(), runner, "smartctl", Device{Name: "/dev/bus/0", Type: "megaraid,1"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Device.ID == second.Device.ID || first.Device.ID == "" || second.Device.ID == "" {
		t.Fatalf("identities overlap: %+v %+v", first.Device, second.Device)
	}
}
