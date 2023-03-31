//go:build windows
// +build windows

package disk

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yusufpapurcu/wmi"
)

type Win32DiskDrive struct {
	Caption      string
	DeviceID     string
	Model        string
	SerialNumber string
	Size         uint64
	PNPDeviceID  string
}

type Win32DiskDriveTemperature struct {
	Active             bool
	Checksum           uint8
	ErrorLogCapability uint8
	//ExtendPollTimeInMinutes  uint8
	InstanceName             string
	Length                   uint32
	OfflineCollectCapability uint8
	OfflineCollectionStatus  uint8
	Reserved                 []uint8
	SelfTestStatus           uint8
	//ShortPollTimeMinutes     uint8
	SmartCapability uint16
	TotalTime       uint16
	VendorSpecific  []uint8
	VendorSpecific2 uint8
	VendorSpecific3 uint8
	VendorSpecific4 []uint8
}

const (
	Power_On_Hours      = 9
	Power_Cycle_Count   = 12
	Temperature_Celsius = 194
)

type SmartInfo struct {
	PowerOnHours       int64
	PowerCycleCount    string
	TemperatureCelsius int8
}

func GetInfo() []*DiskInfo {
	disks := []*DiskInfo{}

	//查询Win32_DiskDrive类信息，获取硬盘信息
	var drives []Win32DiskDrive
	err := wmi.Query("SELECT * FROM Win32_DiskDrive", &drives)
	if err != nil {
		fmt.Println("Query Win32_DiskDrive error:", err)
		panic(err)
	}

	for _, drive := range drives {

		//查询Win32_DiskDriveTemperature类信息，获取硬盘温度
		var temps []Win32DiskDriveTemperature
		PNPDeviceID := strings.ReplaceAll(drive.PNPDeviceID, "\\", "\\\\")
		fmt.Println("PNPDeviceID:", PNPDeviceID)
		queryString := fmt.Sprintf("SELECT * FROM MSStorageDriver_ATAPISmartData WHERE InstanceName LIKE '%%%s%%'", PNPDeviceID)
		// fmt.Println("queryString:", queryString)
		err = wmi.QueryNamespace(queryString, &temps, "root\\WMI")
		if err != nil {
			fmt.Println("Query Win32_DiskDriveTemperature error:", err)
			continue
		}
		if len(temps) < 1 {
			fmt.Println("No temps found")
			continue
		}

		for _, v := range temps {
			result := SplitUintN(v.VendorSpecific[2:], 12)
			if result == nil {
				continue
			}

			smartInfo := parseSmartData(result)

			diskInfo := DiskInfo{}
			diskInfo.ModelName = drive.Model
			diskInfo.SmartStatus = SmartStatus{Passed: v.Active}
			diskInfo.UserCapacity = UserCapacity{Blocks: 0, Bytes: int64(drive.Size)}
			diskInfo.Temperature = Temperature{Current: int8(smartInfo.TemperatureCelsius)}
			diskInfo.PowerOnTime = PowerOnTime{Hours: int64(smartInfo.PowerOnHours)}

			// fmt.Println("InfoName:", path)
			fmt.Println("ModelName:", diskInfo.ModelName)
			fmt.Println("SmartStatus:", diskInfo.SmartStatus.Passed)
			fmt.Println("UserCapacity:", diskInfo.UserCapacity.Bytes)
			fmt.Println("Temperature:", diskInfo.Temperature.Current)
			fmt.Println("PowerOnTime:", diskInfo.PowerOnTime.Hours)
			println("=============")

			disks = append(disks, &diskInfo)
		}
	}

	return disks
}

// 解析smart消息
func parseSmartData(data [][]uint8) *SmartInfo {
	smartInfo := new(SmartInfo)
	for _, v := range data {
		if len(v) != 12 {
			continue
		}
		v5 := int(v[5])
		v6 := int(v[6]) * 16 * 16
		v7 := int(v[7]) * 16 * 16 * 16
		v9 := int(v[9])
		p := v5 + v6 + v7
		switch v[0] {
		case Power_On_Hours:
			smartInfo.PowerOnHours = int64(p)
		case Power_Cycle_Count:
			smartInfo.PowerCycleCount = strconv.Itoa(p)
		case Temperature_Celsius:
			smartInfo.TemperatureCelsius = int8(v9)
		}
	}
	return smartInfo
}

// SplitUintN 按给定的长度进行分割
func SplitUintN(source []uint8, size int) [][]uint8 {
	if len(source) < size || size <= 0 {
		return nil
	}
	result := make([][]uint8, 0)
	for i := 0; i < len(source)/size; i++ {
		if source[i*size] == 0 {
			continue
		}
		if len(source[i*size:]) > size {
			result = append(result, source[i*size:i*size+size])
		} else {
			result = append(result, source[i*size:])
		}
	}
	return result
}
