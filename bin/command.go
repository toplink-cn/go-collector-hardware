package bin

import (
	"bytes"
	"collector/utils"
	"fmt"
	"os/exec"
	"strings"
)

func RunCommand(filename string, args ...string) ([]byte, error) {
	rootDir := utils.GetBinDir()
	osType := utils.GetOsType()
	switch osType {
	case "linux":
		filename = rootDir + "/" + filename
	case "windows":
		filename = rootDir + "\\" + filename
	default:
		fmt.Println("Unknown OS")
	}
	fmt.Println("cmd:", filename+" "+strings.Join(args, " "))
	cmd := exec.Command(filename, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}

	return out, nil
}

func RunCommandAndReturnBytes(filename string, args ...string) bytes.Buffer {
	rootDir := utils.GetBinDir()
	osType := utils.GetOsType()
	switch osType {
	case "linux":
		filename = rootDir + "/" + filename
	case "windows":
		filename = rootDir + "\\" + filename
	default:
		fmt.Println("Unknown OS")
	}
	cmd := exec.Command(filename, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println("Run RunCommandAndReturnBytes Error: ", filename)
	}

	return out
}
