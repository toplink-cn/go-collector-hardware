package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	if IsTesting() {
		fmt.Println("go run")
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
		}
		dir := wd + "/bin"
		switch GetOsType() {
		case "linux":
			dir = dir + "/linux"
		case "widnows":
			dir = dir + "/widnows"
		default:
			fmt.Println("Unknown OS")
		}

		return dir
	} else {
		fmt.Println("exec binary")
		exePath, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exeDir := filepath.Dir(exePath)
		templatesDir := filepath.Join(exeDir, "bin")
		fmt.Println("templatesDir:", templatesDir)
		return templatesDir
	}
}

func IsTesting() bool {
	args := os.Args
	return len(args) > 0 && strings.Contains(strings.ToLower(os.Args[0]), "go-build")
}
