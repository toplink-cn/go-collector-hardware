package main

import (
	"bytes"
	"collector/modules/cpu"
	"collector/modules/disk"
	"collector/modules/ipmi"
	"collector/modules/memory"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	IOCTL_WINRING0_BASE        = 0x911                      // WinRing0 base control code
	IOCTL_WINRING0_RDTSC       = IOCTL_WINRING0_BASE + 0x00 // Read TSC value
	IOCTL_WINRING0_TEMPERATURE = IOCTL_WINRING0_BASE + 0x04 // Read temperature
)

type Ring0IoctlInput struct {
	IoctlCode  uint32
	Input      uintptr
	InputSize  uint32
	Output     uintptr
	OutputSize uint32
}

type Ring0Temperature struct {
	TjMax           uint32
	CoreTemperature uint32
}

func main() {
	cpu.GetInfo()
}

type RespData struct {
	Cpus        cpu.CpuObj
	Disks       []*disk.DiskInfo
	Memory      memory.Memory
	IpmiSensors []*ipmi.Sensor
}

func sendData() {
	url := "http://192.168.88.107:9502"

	data := getInfo()

	jsonData, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Failed to send data:", err)
		return
	}
	defer resp.Body.Close()

	// 获取响应内容
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response:", err)
		return
	}

	fmt.Println("Response:", string(respData))
}

func getInfo() *RespData {
	respData := new(RespData)

	respData.Disks = disk.GetInfo()
	respData.Cpus = cpu.GetInfo()

	respData.Memory = memory.GetInfo()
	respData.IpmiSensors = ipmi.GetInfo()

	return respData
}
