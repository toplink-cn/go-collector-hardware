package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func GetOsType() string {
	if runtime.GOOS == "windows" {
		fmt.Println("Windows")
		return "windows"
	} else if runtime.GOOS == "linux" {
		fmt.Println("Linux")
		return "linux"
	} else {
		fmt.Println("Unknown")
		return "unknown"
	}
}

func GetBinDir() string {
	// 获取当前可执行文件的绝对路径
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// 获取可执行文件所在的目录路径
	exeDir := filepath.Dir(exePath)

	// 将目录路径拼接上您需要的文件夹名字，例如"templates"
	templatesDir := filepath.Join(exeDir, "bin")
	fmt.Println("templatesDir:", templatesDir)
	return templatesDir
}
