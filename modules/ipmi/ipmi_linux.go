//go:build linux
// +build linux

package ipmi

import (
	"bytes"
	"collector/bin"
	"fmt"
)

func GetInfo() []Sensor {

	out := bin.RunCommandAndReturnBytes("ipmitool", "sensor")

	sensors := []*Sensor{}
	// 解析传感器数据
	lines := bytes.Split(out.Bytes(), []byte{'\n'})
	for _, line := range lines {
		fields := bytes.Split(line, []byte{'|'})
		if len(fields) < 2 {
			continue
		}

		sensor := Sensor{Key: fields[0], Value: fields[1]}

		fmt.Printf("Sensor: %s, Value: %s\n", sensor.Key, sensor.Value)

		sensors = append(sensors, sensor)
	}

	return sensors
}
