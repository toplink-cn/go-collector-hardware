package cpu

import (
	// "bufio"
	"collector/utils"
	// "fmt"
	// "io/ioutil"
	// "os"
	// "regexp"
	// "strconv"
	// "strings"
)

func GetInfo() (cpuObj CpuObj) {

	switch utils.GetOsType() {
	case "linux":
		// cpus = getInfoViaLinux()
	case "windows":
		cpuObj = getInfoViaWin()
	default:
	}

	return cpuObj
}

// func getInfoViaLinux() (cpus CpuType) {
// 	usageCpus := getUsage()
// 	temperatureCpus := getTemperature()

// 	cpus.Usage = usageCpus
// 	cpus.Temperature = temperatureCpus

// 	return cpus
// }

// func getUsage() []*CpuObj {
// 	// 获取 CPU 使用率
// 	statData, err := ioutil.ReadFile("/proc/stat")
// 	if err != nil {
// 		panic(err)
// 	}
// 	reader := bufio.NewReader(strings.NewReader(string(statData)))

// 	cpuCores := []*CpuCore{}
// 	for {
// 		line, err := reader.ReadString('\n')
// 		if err != nil {
// 			break
// 		}
// 		fields := strings.Fields(line)
// 		if len(fields) < 5 || !strings.HasPrefix(fields[0], "cpu") {
// 			continue
// 		}

// 		// 获取核心编号
// 		coreID, err := strconv.Atoi(strings.TrimPrefix(fields[0], "cpu"))
// 		if err != nil {
// 			continue
// 		}

// 		user, _ := strconv.Atoi(fields[1])
// 		nice, _ := strconv.Atoi(fields[2])
// 		system, _ := strconv.Atoi(fields[3])
// 		idle, _ := strconv.Atoi(fields[4])
// 		iowait, _ := strconv.Atoi(fields[5])
// 		irq, _ := strconv.Atoi(fields[6])
// 		softirq, _ := strconv.Atoi(fields[7])
// 		steal, _ := strconv.Atoi(fields[8])
// 		// guest, _ := strconv.Atoi(fields[9])
// 		// guestNice, _ := strconv.Atoi(fields[10])

// 		total := user + nice + system + idle + iowait + irq + softirq + steal
// 		usage := float64(total-idle) / float64(total) * 100.0

// 		cpuCore := new(CpuCore)
// 		cpuCore.ID = int8(coreID)
// 		cpuCore.Key = "Usage"
// 		cpuCore.Val = usage

// 		cpuCores = append(cpuCores, cpuCore)
// 		fmt.Printf("Core %d Usage: %.2f%%\n", coreID, usage)
// 	}

// 	cpuObjSlice := []*CpuObj{}

// 	cpuObj := new(CpuObj)
// 	cpuObj.ID = 0
// 	cpuObj.cores = cpuCores

// 	cpuObjSlice = append(cpuObjSlice, cpuObj)

// 	return cpuObjSlice
// }

// func getTemperature() []*CpuObj {
// 	// 获取 CPU 温度
// 	tempPath := ""
// 	tempPath_prefix := ""
// 	if _, err := os.Stat("/sys/class/thermal/thermal_zone0/temp"); err == nil {
// 		// Intel CPU 温度传感器路径
// 		tempPath_prefix = "/sys/class/thermal"
// 		tempPath = tempPath_prefix + "/thermal_zone0/temp"
// 	} else if _, err := os.Stat("/sys/class/hwmon/hwmon0/temp1_input"); err == nil {
// 		// AMD CPU 温度传感器路径
// 		tempPath_prefix = "/sys/class/hwmon"
// 		tempPath = tempPath_prefix + "/hwmon0/temp1_input"
// 	} else {
// 		panic("Unable to find CPU temperature sensor")
// 	}
// 	tempData, err := ioutil.ReadFile(tempPath)
// 	if err != nil {
// 		panic(err)
// 	}
// 	temp, err := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64)
// 	if err != nil {
// 		panic(err)
// 	}
// 	temp = temp / 1000.0 // 转换为摄氏度
// 	fmt.Println(temp)

// 	// 获取 CPU 核心数量
// 	cpuInfoData, err := ioutil.ReadFile("/proc/cpuinfo")
// 	if err != nil {
// 		panic(err)
// 	}
// 	coreCount := 0
// 	threadRegexp := regexp.MustCompile(`^cpu cores\s*:.*$`)
// 	reader := strings.NewReader(string(cpuInfoData))
// 	scanner := bufio.NewScanner(reader)
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if threadRegexp.MatchString(line) {
// 			// fmt.Println("line:", line)
// 			fields := strings.Fields(line)
// 			// fmt.Println("field-len:", len(fields))
// 			if len(fields) > 1 {
// 				coreCount, _ = strconv.Atoi(fields[3])
// 				break
// 			}
// 		}
// 	}
// 	fmt.Println("coreCount:", coreCount)
// 	if coreCount == 0 {
// 		panic("Unable to determine CPU core count")
// 	}

// 	// 获取每个 CPU 核心的温度
// 	cpuIndex := 0
// 	cpuObjs := []*CpuObj{}
// coreOuter:
// 	for num := 0; num <= coreCount; num++ {
// 		cpuCores := []*CpuCore{}
// 		for i := 1; i <= coreCount; i++ {
// 			coreNamePath := fmt.Sprintf("/sys/class/hwmon/hwmon%d/name", num)
// 			coreName, err := ioutil.ReadFile(coreNamePath)
// 			trimmedCoreName := strings.TrimSpace(string(coreName))
// 			if err != nil || !isCpuName(trimmedCoreName) {
// 				continue coreOuter
// 			}
// 			corePath := fmt.Sprintf("/sys/class/hwmon/hwmon%d/temp%d_input", num, i)
// 			coreData, err := ioutil.ReadFile(corePath)
// 			if err != nil {
// 				continue
// 			}
// 			coreLabelPath := fmt.Sprintf("/sys/class/hwmon/hwmon%d/temp%d_label", num, i)
// 			coreLabel, err := ioutil.ReadFile(coreLabelPath)
// 			if err != nil {
// 				continue
// 			}
// 			trimmedCoreLabel := strings.TrimSpace(string(coreLabel))

// 			coreTemp, err := strconv.ParseFloat(strings.TrimSpace(string(coreData)), 64)
// 			if err != nil {
// 				continue
// 			}
// 			coreTemp = coreTemp / 1000.0 // 转换为摄氏度

// 			cpuCore := new(CpuCore)
// 			cpuCore.ID = int8(i)
// 			cpuCore.Label = trimmedCoreLabel
// 			cpuCore.Key = "Temperature"
// 			cpuCore.Val = coreTemp

// 			cpuCores = append(cpuCores, cpuCore)

// 			fmt.Printf("%d - %d - %s Temperature: %.2f C\n", cpuIndex, i, trimmedCoreLabel, coreTemp)
// 		}

// 		cpuObj := new(CpuObj)
// 		cpuObj.ID = int8(cpuIndex)
// 		cpuObj.cores = cpuCores

// 		cpuObjs = append(cpuObjs, cpuObj)

// 		cpuIndex++
// 	}
// 	return cpuObjs
// }

// func isCpuName(name string) bool {
// 	return name == "coretemp" || // Intel CPUs
// 		name == "k10temp" // AMD Ryzen CPUs
// }
