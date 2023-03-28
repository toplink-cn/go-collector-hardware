package disk

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type Smartctl struct {
	Devices []Device `json:"devices"`
}

type Device struct {
	Name     string `json:"name"`
	InfoName string `json:"info_name"`
	Type     string `json:"type"`
	Protocol string `json:"protocol"`
}

type DiskInfo struct {
	ModelName    string       `json:"model_name"`
	SmartStatus  SmartStatus  `json:"smart_status"`
	UserCapacity UserCapacity `json:"user_capacity"`
	Temperature  Temperature  `json:"temperature"`
	PowerOnTime  PowerOnTime  `json:"power_on_time"`
}

type SmartStatus struct {
	Passed bool `json:"passed"`
}

type UserCapacity struct {
	Blocks int64 `json:"blocks"`
	Bytes  int64 `json:"bytes"`
}

type Temperature struct {
	Current int8 `json:"current"`
}

type PowerOnTime struct {
	Hours int64 `json:"hours"`
}

func GetInfo() []*DiskInfo {
	cmd := exec.Command("./bin/smartctl", "--json=c", "--scan")
	output, err := cmd.Output()
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

func getDiskInfo(path string) DiskInfo {
	// 定义要执行的命令和参数
	command := "smartctl"
	args := []string{"--json=c", "-a", path}

	// 执行命令并获取输出结果
	output, err := exec.Command(command, args...).Output()
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
