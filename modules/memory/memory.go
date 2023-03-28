package memory

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"
)

func GetInfo() (*mem.VirtualMemoryStat, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("Failed to get virtual memory info:", err)
		return vm, err
	}

	fmt.Printf("mem total: %v, free: %v \n", vm.Total, vm.Free)

	return vm, err
}
