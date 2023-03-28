package disk

import (
	"collector/bin"
	"collector/utils"
	"encoding/json"
	"fmt"
)

func GetInfo() []*DiskInfo {

	disks := []*DiskInfo{}
	switch utils.GetOsType() {
	case "linux":
		disks = getInfoViaLinux()
		break
	case "windows":
		disks = getInfoViaWindows()
		break
	}

	return disks
}

func getInfoViaLinux() []*DiskInfo {
	output, err := bin.RunCommand("smartctl", "--json=c", "--scan")
	if err != nil {
		panic(err)
	}

	var s Smartctl
	if err := json.Unmarshal([]byte(output), &s); err != nil {
		panic(err)
	}

	// 获取 devices 信息
	disks := []*DiskInfo{}
	for _, d := range s.Devices {
		diskInfo := getDiskInfo(d.InfoName)
		fmt.Println("InfoName:", d.InfoName)
		fmt.Println("ModelName:", diskInfo.ModelName)
		fmt.Println("SmartStatus:", diskInfo.SmartStatus.Passed)
		fmt.Println("UserCapacity:", diskInfo.UserCapacity.Bytes)
		fmt.Println("Temperature:", diskInfo.Temperature.Current)
		fmt.Println("PowerOnTime:", diskInfo.PowerOnTime.Hours)
		println("=============")
		disks = append(disks, &diskInfo)
	}
	return disks
}

func getInfoViaWindows() []*DiskInfo {
	disks := []*DiskInfo{}

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

	return diskInfo
}
