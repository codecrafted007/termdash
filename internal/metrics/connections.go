package metrics

import (
	psnet "github.com/shirou/gopsutil/v3/net"
)

// CollectConnectionCounts returns a map of PID → number of network connections.
// Returns an empty map on error (graceful degradation — no root = no data on some OSes).
func CollectConnectionCounts() map[int32]int {
	conns, err := psnet.Connections("all")
	if err != nil {
		return map[int32]int{}
	}

	counts := make(map[int32]int)
	for _, c := range conns {
		if c.Pid > 0 {
			counts[int32(c.Pid)]++
		}
	}
	return counts
}
