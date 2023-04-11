package utils

import (
	"fmt"
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
	_, filename, _, _ := runtime.Caller(0)

	rootPath := filepath.Join(filepath.Dir(filename), "../")
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		panic(err)
	}
	return absPath + "/bin"
}
