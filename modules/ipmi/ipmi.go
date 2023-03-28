package ipmi

import (
	"bytes"
	"collector/bin"
	"fmt"
)

type Sensor struct {
	Key   string
	Value string
}

func GetInfo() []*Sensor {

	// 执行 ipmitool.static 命令
	// cmd := exec.Command("./bin/ipmitool.static", "sensor")
	// var out bytes.Buffer
	// cmd.Stdout = &out
	// err := cmd.Run()
	// if err != nil {
	// 	panic(err)
	// }
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
