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
		diskInfo := getDiskInfo(d.InfoName)
		fmt.Println("InfoName:", d.InfoName)
		fmt.Println("ModelName:", diskInfo.ModelName)
		fmt.Println("SerialNumber:", diskInfo.SerialNumber)
		fmt.Println("ModelFamily:", diskInfo.ModelFamily)
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

	output, err := bin.RunCommand("smartctl", args...)
	if err != nil {
		panic(err)
	}

	var diskInfo DiskInfo
	if err := json.Unmarshal([]byte(output), &diskInfo); err != nil {
		fmt.Println("Failed to unmarshal JSON:", err)
		return diskInfo
	}

	modelFamily := strings.ToLower(diskInfo.ModelFamily)
	if strings.Contains(modelFamily, "ssd") {
		diskInfo.ModelType = "ssd"
	} else if strings.Contains(modelFamily, "hhd") {
		diskInfo.ModelType = "hdd"
	} else if strings.Contains(modelFamily, "nvme") {
		diskInfo.ModelType = "nvme"
	} else {
		diskInfo.ModelType = "unknown"
	}

	return diskInfo
}
