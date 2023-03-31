package memory

import (
	"fmt"

	"github.com/yusufpapurcu/wmi"
)

type Win32_OperatingSystem struct {
	FreePhysicalMemory     uint64
	TotalVisibleMemorySize uint64
}

func getInfoViaWin() (memory Memory) {
	var os []Win32_OperatingSystem
	err := wmi.QueryNamespace("SELECT FreePhysicalMemory, TotalVisibleMemorySize FROM Win32_OperatingSystem", &os, "ROOT\\CIMV2")
	if err != nil {
		panic(err)
	}

	if len(os) < 1 {
		return Memory{}
	}

	for _, item := range os {
		memory.Total = int64(item.TotalVisibleMemorySize)
		memory.Free = int64(item.FreePhysicalMemory)
		break
	}
	fmt.Printf("mem total: %v, free: %v \n", memory.Total, memory.Free)

	return memory
}
