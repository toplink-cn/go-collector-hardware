package ipmi

import (
	"bytes"
	"collector/bin"
	"collector/utils"
	"fmt"
)

type Sensor struct {
	Key   string
	Value string
}

func GetInfo() []*Sensor {
	sensors := []*Sensor{}

	switch utils.GetOsType() {
	case "linux":
		sensors = getInfoViaLinux()
	case "windows":
		sensors = getInfoViaWin()
	default:
	}

	return sensors
}

func getInfoViaLinux() []*Sensor {

	out := bin.RunCommandAndReturnBytes("ipmitool", "sensor")

	sensors := []*Sensor{}
	// 解析传感器数据
	lines := bytes.Split(out.Bytes(), []byte{'\n'})
	for _, line := range lines {
		fields := bytes.Split(line, []byte{'|'})
		if len(fields) < 2 {
			continue
		}

		sensor := new(Sensor)
		sensor.Key = string(fields[0])
		sensor.Value = string(fields[1])
		fmt.Printf("Sensor: %s, Value: %s\n", sensor.Key, sensor.Value)

		sensors = append(sensors, sensor)
	}

	return sensors
}
