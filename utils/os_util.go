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
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	absPath, err := filepath.Abs(wd)
	if err != nil {
		panic(err)
	}
	return absPath + "/bin"
}
