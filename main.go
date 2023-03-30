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
	"syscall"
	"unsafe"
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
	var kernel32 = syscall.MustLoadDLL("kernel32.dll")
	var procGetCurrentProcessId = kernel32.MustFindProc("GetCurrentProcessId")
	fmt.Println("processId:", procGetCurrentProcessId)

	var dll = syscall.MustLoadDLL("WinRing0x64.dll")

	// Get the function pointer for the Ring0Control() function
	proc, err := dll.FindProc("Ring0Control")
	if err != nil {
		fmt.Println("Failed to find Ring0Control() function:", err)
		return
	}

	// Initialize the input and output structures
	input := Ring0IoctlInput{
		IoctlCode:  IOCTL_WINRING0_TEMPERATURE,
		Input:      0,
		InputSize:  0,
		Output:     uintptr(unsafe.Pointer(&Ring0Temperature{})),
		OutputSize: uint32(unsafe.Sizeof(Ring0Temperature{})),
	}

	var output Ring0Temperature

	// Call the Ring0Control() function to get the temperature information
	_, _, err = proc.Call(uintptr(unsafe.Pointer(&input)), uintptr(unsafe.Pointer(&output)), 0)
	if err != nil {
		fmt.Println("Failed to get CPU temperature:", err)
		return
	}

	// Calculate the temperature for each core
	for i := 0; i < 64; i++ {
		temp := float32(output.CoreTemperature >> uint(i) & 0xFF)
		if temp == 0 {
			break
		}

		fmt.Printf("Core %d temperature: %f°C\n", i, temp)
	}
}

type RespData struct {
	Cpus        cpu.CpuType
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
