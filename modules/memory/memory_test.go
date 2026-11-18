package memory

import (
	"github.com/shirou/gopsutil/v3/mem"
	"math"
	"testing"
)

func TestMemoryUsesAvailableBytesInsteadOfUnreclaimableFree(t *testing.T) {
	point, err := fromMemory(&mem.VirtualMemoryStat{Total: 16 * 1024 * 1024 * 1024, Free: 1024, Available: 8 * 1024 * 1024 * 1024})
	if err != nil || point.Total != 16*1024*1024*1024 || point.Free != 8*1024*1024*1024 {
		t.Fatalf("%+v %v", point, err)
	}
}

func TestInvalidMemoryDoesNotLookLikeZeroPercentUsed(t *testing.T) {
	for _, point := range []*mem.VirtualMemoryStat{nil, {}, {Total: 100, Available: 101}, {Total: math.MaxUint64}} {
		if _, err := fromMemory(point); err == nil {
			t.Fatalf("accepted invalid memory %+v", point)
		}
	}
}
