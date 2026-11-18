package network

import (
	gonet "github.com/shirou/gopsutil/v3/net"
	"math"
	"testing"
)

func TestNetworkReturnsCumulativeByteCountersWithStableInterfaceNames(t *testing.T) {
	points, err := fromCounters([]gonet.IOCountersStat{{Name: "eth1", BytesRecv: 0, BytesSent: 456}, {Name: "eth0", BytesRecv: 123, BytesSent: 789}})
	if err != nil || len(points) != 2 || points[0].Interface != "eth0" || points[0].Receive != 123 || points[1].Receive != 0 || points[1].Transmit != 456 {
		t.Fatalf("%+v %v", points, err)
	}
}
func TestInvalidOrDuplicateCountersRemainPartial(t *testing.T) {
	points, err := fromCounters([]gonet.IOCountersStat{{Name: "eth0", BytesRecv: 1}, {Name: "eth0", BytesRecv: 2}, {Name: "eth1", BytesSent: math.MaxUint64}})
	if len(points) != 1 || err == nil {
		t.Fatalf("%+v %v", points, err)
	}
}
