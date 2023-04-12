package bin

import (
	"bytes"
	"collector/utils"
	"fmt"
	"os/exec"
)

func RunCommand(filename string, args ...string) ([]byte, error) {
	rootDir := utils.GetBinDir()
	filename = rootDir + "/" + filename
	fmt.Println("filename:", filename)
	cmd := exec.Command(filename, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func RunCommandAndReturnBytes(filename string, args ...string) bytes.Buffer {
	rootDir := utils.GetBinDir()
	filename = rootDir + "/" + filename
	cmd := exec.Command(filename, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println("Run RunCommandAndReturnBytes Error: ", filename)
	}

	return out
}
