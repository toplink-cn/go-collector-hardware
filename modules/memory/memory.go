package memory

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/mem"
)

type Memory struct {
	Total int64
	Free  int64
}

func GetInfo() *Memory {
	vm, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("Failed to get virtual memory info:", err)
		panic(err)
	}

	fmt.Printf("mem total: %v, free: %v \n", vm.Total, vm.Free)

	memory := new(Memory)
	memory.Total = int64(vm.Total)
	memory.Free = int64(vm.Free)

	return memory
}
