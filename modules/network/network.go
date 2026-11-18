package network

import (
	"collector/modules/collection"
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	gonet "github.com/shirou/gopsutil/v3/net"
)

// Counters are monotonically cumulative bytes per named interface, never rates.
type InterfaceCounter struct {
	Interface string
	Receive   uint64
	Transmit  uint64
}

func Collect(ctx context.Context) (collection.Result[[]InterfaceCounter], error) {
	if err := ctx.Err(); err != nil {
		return collection.Result[[]InterfaceCounter]{Quality: map[string]collection.Quality{"network": collection.FromError(err, false)}}, err
	}
	points, err := gonet.IOCountersWithContext(ctx, true)
	values := []InterfaceCounter{}
	if err == nil {
		values, err = fromCounters(points)
	}
	return collection.Result[[]InterfaceCounter]{Data: values, Quality: map[string]collection.Quality{"network": collection.FromError(err, len(values) > 0)}}, err
}

func fromCounters(points []gonet.IOCountersStat) ([]InterfaceCounter, error) {
	values := []InterfaceCounter{}
	seen := map[string]bool{}
	problems := []error{}
	for _, point := range points {
		if strings.TrimSpace(point.Name) == "" || seen[point.Name] || point.BytesRecv > math.MaxInt64 || point.BytesSent > math.MaxInt64 {
			problems = append(problems, fmt.Errorf("invalid network interface observation %q", point.Name))
			continue
		}
		seen[point.Name] = true
		values = append(values, InterfaceCounter{Interface: point.Name, Receive: point.BytesRecv, Transmit: point.BytesSent})
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Interface < values[j].Interface })
	if len(values) == 0 {
		problems = append(problems, collection.ErrUnsupported)
	}
	return values, errors.Join(problems...)
}

func GetInfo() []InterfaceCounter { result, _ := Collect(context.Background()); return result.Data }
