//go:build linux
// +build linux

package memory

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"
)

func GetInfo() Memory {
	vm, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("Failed to get virtual memory info:", err)
		panic(err)
	}

	fmt.Printf("mem total: %v, free: %v \n", vm.Total, vm.Free)

	memory := Memory{Total: int64(vm.Total), Free: int64(vm.Free)}

	return memory
}
