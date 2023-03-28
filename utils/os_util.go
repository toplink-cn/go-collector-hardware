package utils

import (
	"fmt"
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
