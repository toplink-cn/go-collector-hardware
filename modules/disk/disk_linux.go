//go:build linux
// +build linux

package disk

import (
	"collector/bin"
	"encoding/json"
	"fmt"
	"strings"
)

func GetInfo() []DiskInfo {
	output, err := bin.RunCommand("smartctl", "--json=c", "--scan")
	if err != nil {
		panic(err)
	}

	var s Smartctl
	if err := json.Unmarshal([]byte(output), &s); err != nil {
		panic(err)
	}

	disks := []DiskInfo{}
	for _, d := range s.Devices {
		args := []string{"--json=c", "-a", d.InfoName, "-d", d.Type}
		diskInfo := getDiskInfo(d.InfoName, args...)

		if diskInfo.ModelName == "" {
			continue
		}

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

func getDiskInfo(path string, args ...string) DiskInfo {
	output, err := bin.RunCommand("smartctl", args...)
	if err != nil {
		fmt.Println("getDiskInfo err:", err.Error())
	}

	var diskInfo DiskInfo
	if err := json.Unmarshal([]byte(output), &diskInfo); err != nil {
		fmt.Println("Failed to unmarshal JSON:", err)
		return DiskInfo{}
	}

	if diskInfo.ScsiVendor == "LSI" {
		return DiskInfo{}
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
