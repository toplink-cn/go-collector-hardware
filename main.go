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
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// 获取可执行文件的绝对路径
	exePath, err := filepath.Abs(os.Args[0])
	if err != nil {
		fmt.Println("获取可执行文件路径失败:", err)
		return
	}

	// 创建文件锁
	lockFile, err := os.Create(exePath + ".lock")
	if err != nil {
		fmt.Println("创建文件锁失败:", err)
		return
	}

	// 尝试获取独占锁
	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		fmt.Println("只能运行一个实例")
		return
	}
	sendData()

	// 关闭文件锁
	err = lockFile.Close()
	if err != nil {
		fmt.Println("关闭文件锁失败:", err)
	}
}

type RespData struct {
	Cpus        cpu.CpuObj
	Disks       []disk.DiskInfo
	Memory      memory.Memory
	IpmiSensors []ipmi.Sensor
}

func sendData() {
	// 加载.env文件
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	// url := "http://192.168.88.107:9502"
	// 从环境变量中获取host
	url := os.Getenv("HOST")
	fmt.Println("url:", url)

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
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02 15:04:05")
	fmt.Println(formattedTime+", Response:", string(respData))
}

func getInfo() *RespData {
	respData := new(RespData)

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		respData.Disks = disk.GetInfo()
		wg.Done()
	}()

	go func() {
		respData.Cpus = cpu.GetInfo()
		wg.Done()
	}()

	go func() {
		respData.Memory = memory.GetInfo()
		wg.Done()
	}()

	go func() {
		respData.IpmiSensors = ipmi.GetInfo()
		wg.Done()
	}()

	wg.Wait()

	return respData
}
