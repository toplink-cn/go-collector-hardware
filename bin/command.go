package bin

import (
	"bytes"
	"embed"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

//go:embed smartctl ipmitool OpenHardwareMonitorReport.exe OpenHardwareMonitorLib.dll
var fs embed.FS

// 执行命令
func RunCommand(cmdName string, args ...string) ([]byte, error) {
	// 读取命令文件的内容
	data, err := fs.ReadFile(cmdName)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %s from embed FS: %v", cmdName, err)
	}

	// 创建一个临时文件并将命令文件的内容写入到该文件中
	file, err := ioutil.TempFile("", cmdName)
	if err != nil {
		return nil, fmt.Errorf("Failed to create temp file for %s: %v", cmdName, err)
	}
	defer os.Remove(file.Name())

	if _, err := file.Write(data); err != nil {
		return nil, fmt.Errorf("Failed to write %s to temp file: %v", cmdName, err)
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("Failed to close temp file for %s: %v", cmdName, err)
	}

	// 设置命令文件的执行权限
	if err := os.Chmod(file.Name(), 0700); err != nil {
		return nil, fmt.Errorf("Failed to set executable permission for %s: %v", cmdName, err)
	}

	// 执行命令
	cmd := exec.Command(file.Name(), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute %s: %v", cmdName, err)
	}

	return out, nil
}

// 执行命令
func RunCommandAndReturnBytes(cmdName string, args ...string) bytes.Buffer {
	// 读取命令文件的内容
	data, err := fs.ReadFile(cmdName)
	if err != nil {
		fmt.Println("can not read file")
		panic(err)
	}

	// 创建一个临时文件并将命令文件的内容写入到该文件中
	file, err := ioutil.TempFile("", cmdName)
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	if _, err := file.Write(data); err != nil {
		panic(err)
	}

	if err := file.Close(); err != nil {
		panic(err)
	}

	// 设置命令文件的执行权限
	if err := os.Chmod(file.Name(), 0700); err != nil {
		panic(err)
	}
	fmt.Println("temp filename:", file.Name())
	// 执行命令
	cmd := exec.Command(file.Name(), args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	return out
}

// 执行命令
func RunOpenHardwareCommand() bytes.Buffer {
	// 创建临时目录
	tmpDir, err := ioutil.TempDir("", "collector")
	if err != nil {
		fmt.Println("Failed to create temporary directory:", err)
		panic(err)
	}
	defer os.Remove(tmpDir)

	openHardware := createTmpFile(tmpDir, "OpenHardwareMonitorReport.exe")
	// fmt.Println("temp filename:", openHardware)
	defer os.Remove(openHardware)

	dll := createTmpFile(tmpDir, "OpenHardwareMonitorLib.dll")
	// fmt.Println("temp filename:", dll)
	defer os.Remove(dll)

	// 执行命令
	cmd := exec.Command(openHardware)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	return out
}

func createTmpFile(tmpDir string, filename string) string {
	data, err := fs.ReadFile(filename)
	if err != nil {
		fmt.Println("can not read file")
		panic(err)
	}

	// 创建临时文件
	tmpFile := fmt.Sprintf("%s\\%s", tmpDir, filename)
	err = ioutil.WriteFile(tmpFile, data, 0700)
	if err != nil {
		fmt.Println("Failed to write data to file:", err)
		panic(err)
	}
	// fmt.Println("write data to file succeed")

	return tmpFile
}
