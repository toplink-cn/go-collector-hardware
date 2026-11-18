package disk

import (
	"collector/bin"
	"collector/modules/collection"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Smartctl struct {
	Devices []Device `json:"devices"`
}
type Device struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name"`
	InfoName string `json:"info_name"`
	Type     string `json:"type"`
	Protocol string `json:"protocol"`
}
type DiskInfo struct {
	ModelName         string       `json:"model_name"`
	SmartStatus       SmartStatus  `json:"smart_status"`
	UserCapacity      UserCapacity `json:"user_capacity"`
	NVMeTotalCapacity *uint64      `json:"nvme_total_capacity,omitempty"`
	Temperature       Temperature  `json:"temperature"`
	PowerOnTime       PowerOnTime  `json:"power_on_time"`
	SerialNumber      string       `json:"serial_number"`
	RotationRate      interface{}  `json:"rotation_rate,omitempty"`
	Device            Device       `json:"device"`
	SetaVersion       SetaVersion  `json:"sata_version"`
	ScsiVendor        string       `json:"scsi_vendor"`
	Product           string       `json:"product,omitempty"`
	ModelType         string
}
type SetaVersion struct {
	String string `json:"string"`
	Value  int64  `json:"value"`
}
type SmartStatus struct {
	Passed *bool `json:"passed"`
}
type UserCapacity struct {
	Blocks *uint64 `json:"blocks"`
	Bytes  *uint64 `json:"bytes"`
}
type Temperature struct {
	Current *float64 `json:"current"`
}
type PowerOnTime struct {
	Hours *uint64 `json:"hours"`
}

type commandRunner func(context.Context, string, ...string) ([]byte, error)

func Collect(ctx context.Context) (collection.Result[[]DiskInfo], error) {
	values, err := collectDisks(ctx, bin.RunCommandContext, smartctlExecutable(), 4)
	return collection.Result[[]DiskInfo]{Data: values, Quality: map[string]collection.Quality{"disks": collection.FromError(err, len(values) > 0)}}, err
}
func GetInfo() []DiskInfo { result, _ := Collect(context.Background()); return result.Data }

func collectDisks(ctx context.Context, run commandRunner, executable string, workers int) ([]DiskInfo, error) {
	output, err := run(ctx, executable, "--json=c", "--scan")
	if err != nil {
		return nil, fmt.Errorf("disk scan: %w", err)
	}
	var scan Smartctl
	if err := json.Unmarshal(output, &scan); err != nil {
		return nil, fmt.Errorf("disk scan JSON: %w", err)
	}
	if len(scan.Devices) == 0 {
		return []DiskInfo{}, collection.ErrUnsupported
	}
	if workers < 1 {
		workers = 1
	}
	if workers > len(scan.Devices) {
		workers = len(scan.Devices)
	}
	type job struct {
		index  int
		device Device
	}
	type observation struct {
		value DiskInfo
		err   error
		done  bool
	}
	observations := make([]observation, len(scan.Devices))
	jobs := make(chan job)
	var group sync.WaitGroup
	for index := 0; index < workers; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for item := range jobs {
				value, err := readDisk(ctx, run, executable, item.device)
				observations[item.index] = observation{value: value, err: err, done: true}
			}
		}()
	}
dispatch:
	for index, device := range scan.Devices {
		select {
		case jobs <- job{index: index, device: device}:
		case <-ctx.Done():
			break dispatch
		}
	}
	close(jobs)
	group.Wait()
	values := []DiskInfo{}
	problems := []error{}
	seen := map[string]bool{}
	for index, observation := range observations {
		if !observation.done {
			problems = append(problems, ctx.Err())
			continue
		}
		identity := deviceIdentity(observation.value.Device)
		if identity == "" {
			problems = append(problems, fmt.Errorf("disk %d has no device identity", index))
			continue
		}
		if seen[identity] {
			problems = append(problems, fmt.Errorf("duplicate disk device %q", identity))
			continue
		}
		seen[identity] = true
		values = append(values, observation.value)
		if observation.err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", identity, observation.err))
		}
	}
	sort.Slice(values, func(i, j int) bool { return deviceIdentity(values[i].Device) < deviceIdentity(values[j].Device) })
	return values, errors.Join(problems...)
}

func deviceIdentity(device Device) string {
	if device.ID != "" {
		return device.ID
	}
	name := device.Name
	if name == "" {
		name = device.InfoName
	}
	if name == "" {
		return ""
	}
	if device.Type != "" {
		name += "|" + device.Type
	}
	return name
}

func readDisk(ctx context.Context, run commandRunner, executable string, device Device) (DiskInfo, error) {
	path := device.Name
	if path == "" {
		path = device.InfoName
	}
	device.ID = deviceIdentity(device)
	value := DiskInfo{Device: device}
	if path == "" {
		return value, errors.New("missing disk device path")
	}
	args := []string{"--json=c", "-a", path}
	if device.Type != "" {
		args = append(args, "-d", device.Type)
	}
	output, commandErr := run(ctx, executable, args...)
	if err := json.Unmarshal(output, &value); err != nil {
		return DiskInfo{Device: device}, fmt.Errorf("disk JSON: %w", errors.Join(commandErr, err))
	}
	if value.Device.Name == "" {
		value.Device.Name = path
	}
	if value.Device.InfoName == "" {
		value.Device.InfoName = device.InfoName
	}
	if value.Device.Type == "" {
		value.Device.Type = device.Type
	}
	value.Device.ID = device.ID
	if value.Device.Protocol == "" {
		value.Device.Protocol = device.Protocol
	}
	problems := []error{}
	if commandErr != nil {
		code, ok := bin.ExitCode(commandErr)
		// smartctl bits 3..7 report observed disk health/errors, not command failure.
		if !ok || code&7 != 0 {
			problems = append(problems, commandErr)
			value.SmartStatus.Passed = nil
		}
	}
	var evidence struct {
		Smartctl struct {
			ExitStatus int `json:"exit_status"`
		} `json:"smartctl"`
	}
	if json.Unmarshal(output, &evidence) == nil && evidence.Smartctl.ExitStatus&7 != 0 {
		problems = append(problems, fmt.Errorf("smartctl command status %d", evidence.Smartctl.ExitStatus))
		value.SmartStatus.Passed = nil
	}
	if value.UserCapacity.Bytes == nil {
		value.UserCapacity.Bytes = value.NVMeTotalCapacity
	}
	if value.UserCapacity.Bytes != nil && (*value.UserCapacity.Bytes == 0 || *value.UserCapacity.Bytes > math.MaxInt64) {
		value.UserCapacity.Bytes = nil
	}
	if value.Temperature.Current != nil {
		temperature := *value.Temperature.Current
		if math.IsNaN(temperature) || math.IsInf(temperature, 0) || temperature <= 0 || temperature > 200 {
			value.Temperature.Current = nil
		}
	}
	if value.PowerOnTime.Hours != nil && *value.PowerOnTime.Hours > math.MaxInt64 {
		value.PowerOnTime.Hours = nil
	}
	if value.ModelName == "" {
		value.ModelName = value.Product
	}
	protocol := value.Device.Protocol
	if value.SetaVersion.String != "" {
		protocol = value.SetaVersion.String
	}
	media := ""
	switch rotation := value.RotationRate.(type) {
	case float64:
		if rotation == 0 {
			media = "ssd"
		} else if rotation > 0 {
			media = "hdd"
		}
	case string:
		if strings.Contains(strings.ToLower(rotation), "solid") {
			media = "ssd"
		} else if rpm, err := strconv.Atoi(rotation); err == nil && rpm > 0 {
			media = "hdd"
		}
	}
	value.ModelType = strings.TrimSpace(protocol + " " + media)
	if value.UserCapacity.Bytes == nil {
		problems = append(problems, errors.New("disk capacity unavailable"))
	}
	if value.SmartStatus.Passed == nil {
		problems = append(problems, errors.New("SMART health unavailable"))
	}
	return value, errors.Join(problems...)
}
