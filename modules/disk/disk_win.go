//go:build windows
// +build windows

package disk

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func GetInfo() []DiskInfo {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	smartctlPath := wd + "\\bin\\smartctl\\smartctl.exe"
	cmd := exec.Command(smartctlPath, "--json=c", "--scan")
	output, err := cmd.CombinedOutput()
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
	// 定义要执行的命令和参数
	args := []string{"--json=c", "-a", path}

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	smartctlPath := wd + "\\bin\\smartctl\\smartctl.exe"
	cmd := exec.Command(smartctlPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
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
