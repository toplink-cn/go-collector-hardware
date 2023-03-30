package memory

import (
	"collector/utils"
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"
)

type Memory struct {
	Total int64
	Free  int64
}

func GetInfo() Memory {

	memory := Memory{}
	switch utils.GetOsType() {
	case "linux":
		memory = getInfoViaLinux()
	case "windows":
		memory = getInfoViaWin()
	default:
	}

	return memory
}

func getInfoViaLinux() Memory {
	vm, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("Failed to get virtual memory info:", err)
		panic(err)
	}

	fmt.Printf("mem total: %v, free: %v \n", vm.Total, vm.Free)

	memory := Memory{Total: int64(vm.Total), Free: int64(vm.Free)}

	return memory
}
