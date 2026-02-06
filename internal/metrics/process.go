package metrics

import (
	"sort"

	"github.com/shirou/gopsutil/v3/process"
)

// collectProcesses returns the top N processes sorted by CPU% descending.
func collectProcesses(topN int) ([]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var infos []ProcessInfo
	for _, p := range procs {
		name, err := p.Name()
		if err != nil {
			continue
		}
		cpuPct, err := p.CPUPercent()
		if err != nil {
			continue
		}
		memPct, err := p.MemoryPercent()
		if err != nil {
			continue
		}
		user, _ := p.Username()

		var rss uint64
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			rss = mi.RSS
		}

		infos = append(infos, ProcessInfo{
			PID:        p.Pid,
			Name:       name,
			CPUPercent: cpuPct,
			MemPercent: memPct,
			User:       user,
			MemRSS:     rss,
			ConnCount:  -1, // populated separately by connection collector
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CPUPercent > infos[j].CPUPercent
	})

	if len(infos) > topN {
		infos = infos[:topN]
	}
	return infos, nil
}
