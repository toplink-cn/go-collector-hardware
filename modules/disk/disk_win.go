//go:build windows
// +build windows

package disk

import (
	"collector/bin"
	"encoding/json"
	"fmt"
	"strings"
)

func GetInfo() []DiskInfo {
	args := []string{"--json=c", "--scan"}
	output, err := bin.RunCommand("smartctl\\smartctl.exe", args...)
	if err != nil {
		panic(err)
	}

	var s Smartctl
	if err := json.Unmarshal([]byte(output), &s); err != nil {
		panic(err)
	}

	disks := []DiskInfo{}
	for _, d := range s.Devices {
		diskInfo := getDiskInfo(d.InfoName)
		fmt.Println("InfoName:", d.InfoName)
		fmt.Println("ModelName:", diskInfo.ModelName)
		fmt.Println("SerialNumber:", diskInfo.SerialNumber)
		fmt.Println("ModelType:", diskInfo.ModelType)
		fmt.Println("SmartStatus:", diskInfo.SmartStatus.Passed)
		fmt.Println("UserCapacity:", diskInfo.UserCapacity.Bytes)
		fmt.Println("Temperature:", diskInfo.Temperature.Current)
		fmt.Println("PowerOnTime:", diskInfo.PowerOnTime.Hours)
		println("=============")
		disks = append(disks, diskInfo)
	}
	return disks
}

func getDiskInfo(path string) DiskInfo {
	args := []string{"--json=c", "-a", path}
	output, err := bin.RunCommand("smartctl\\smartctl.exe", args...)
	if err != nil {
		fmt.Println("err:", err.Error())
	}

	var s Smartctl
	if err := json.Unmarshal([]byte(output), &s); err != nil {
		panic(err)
	}

	var diskInfo DiskInfo
	if err := json.Unmarshal([]byte(output), &diskInfo); err != nil {
		fmt.Println("Failed to unmarshal JSON:", err)
		return diskInfo
	}

	protocol := ""
	modelType := ""
	if diskInfo.SetaVersion.String != "" {
		protocol = diskInfo.SetaVersion.String
	} else {
		if strings.HasPrefix(path, "/dev/sd") {
			protocol = "SATA"
		} else {
			protocol = diskInfo.Device.Protocol
			switch protocol {
			case "NVMe":
				modelType = "NVMe"
			default:
			}
		}
	}
	fmt.Println("RotationRate:", diskInfo.RotationRate)

	if diskInfo.RotationRate != nil {
		if diskInfo.RotationRate == 0 {
			modelType = "ssd"
		} else {
			modelType = "hdd"
		}
	} else {
		modelType = "ssd"
	}

	diskInfo.ModelType = protocol + " " + modelType

	return diskInfo
}
