//go:build windows
// +build windows

package cpu

import (
	"bytes"
	"collector/bin"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yusufpapurcu/wmi"
)

func GetInfo() (cpuObj CpuObj) {

	usageCpus := getWinUsage()
	cpuObj.Usage = usageCpus

	maptempCpu := getWinTemperature()
	cpuObj.Temperature = maptempCpu

	return cpuObj
}

func getWinTemperature() (cpus []CpuAttr) {
	out := bin.RunCommandAndReturnBytes("OpenHardwareMonitorReport.exe")
	lines := bytes.Split(out.Bytes(), []byte{'\n'})
	// 定义正则表达式
	re := regexp.MustCompile(`CPU Core #(\d+)\s+:\s+(\d+)\s+(\d+)\s+(\d+) \(.*\/(\d+)\/temperature\/\d+\)`)

	for _, line := range lines {

		// 进行匹配
		matches := re.FindStringSubmatch(string(line))

		// 检查匹配结果
		if len(matches) > 0 {
			// 打印匹配结果
			_coreId, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			_id := matches[5] + "-" + strconv.Itoa(_coreId-1)

			fmt.Printf("Cpu Temperature %s : %s \n", _id, matches[2])

			cpuAttr := CpuAttr{ID: matches[5] + "-" + matches[1], Value: matches[2]}
			cpus = append(cpus, cpuAttr)
		}
	}

	fmt.Println(cpus)
	return cpus
}

type ProcessorInformation struct {
	Name                 string
	PercentProcessorTime uint64
	PercentIdleTime      uint64
}

func getWinUsage() (cpus []CpuAttr) {

	var processorInformations []ProcessorInformation
	err := wmi.QueryNamespace("SELECT Name,PercentProcessorTime,PercentIdleTime FROM Win32_PerfFormattedData_Counters_ProcessorInformation WHERE NOT Name LIKE '%_Total'", &processorInformations, "ROOT\\CIMV2")
	if err != nil {
		panic(err)
	}
	fmt.Println("len:", len(processorInformations))

	if len(processorInformations) < 1 {
		return cpus
	}

	for _, item := range processorInformations {

		ids := strings.Split(item.Name, ",")
		if len(ids) < 2 {
			continue
		}
		_id := ids[0] + "-" + ids[1]
		_value := strconv.Itoa(int(item.PercentProcessorTime))

		_cpuAttr := CpuAttr{ID: _id, Value: _value}
		fmt.Printf("Cpu Usage %s : %s \n", _id, _value)
		cpus = append(cpus, _cpuAttr)
	}
	fmt.Println("cpus: ", cpus)

	return cpus
}
