package memory

import (
	"collector/modules/collection"
	"context"
	"fmt"
	"math"

	"github.com/shirou/gopsutil/v3/mem"
)

// Total and Free are bytes. Free is reclaimable Available memory, not MemFree.
// gopsutil performs the Windows KiB-to-byte conversion at the OS boundary.
type Memory struct {
	Total int64
	Free  int64
}

func Collect(ctx context.Context) (collection.Result[Memory], error) {
	if err := ctx.Err(); err != nil {
		return collection.Result[Memory]{Quality: map[string]collection.Quality{"memory": collection.FromError(err, false)}}, err
	}
	point, err := mem.VirtualMemoryWithContext(ctx)
	var value Memory
	if err == nil {
		value, err = fromMemory(point)
	}
	return collection.Result[Memory]{Data: value, Quality: map[string]collection.Quality{"memory": collection.FromError(err, false)}}, err
}

func fromMemory(point *mem.VirtualMemoryStat) (Memory, error) {
	if point == nil || point.Total == 0 || point.Total > math.MaxInt64 || point.Available > point.Total {
		return Memory{}, fmt.Errorf("invalid memory observation")
	}
	return Memory{Total: int64(point.Total), Free: int64(point.Available)}, nil
}

func GetInfo() Memory { result, _ := Collect(context.Background()); return result.Data }
