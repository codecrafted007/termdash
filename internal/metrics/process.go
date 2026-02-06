package metrics

import (
	"sort"

	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessCollector caches gopsutil Process objects between ticks so that
// Percent(0) can compute delta-based CPU% instead of lifetime averages.
type ProcessCollector struct {
	cache    map[int32]*process.Process
	totalMem uint64
}

// NewProcessCollector returns an initialized ProcessCollector.
func NewProcessCollector() *ProcessCollector {
	return &ProcessCollector{
		cache: make(map[int32]*process.Process),
	}
}

// CollectAll returns info for ALL running processes (no topN limit).
// Cached *Process objects are reused so that Percent(0) can compute
// accurate delta-based CPU%.
func (pc *ProcessCollector) CollectAll() ([]ProcessInfo, error) {
	// Lazily fetch total memory once.
	if pc.totalMem == 0 {
		if vm, err := mem.VirtualMemory(); err == nil {
			pc.totalMem = vm.Total
		}
	}

	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	// Build a set of live PIDs for pruning the cache afterwards.
	live := make(map[int32]struct{}, len(pids))

	infos := make([]ProcessInfo, 0, len(pids))
	for _, pid := range pids {
		live[pid] = struct{}{}

		p, ok := pc.cache[pid]
		if !ok {
			p, err = process.NewProcess(pid)
			if err != nil {
				continue // process exited between Pids() and here
			}
			pc.cache[pid] = p
		}

		name, err := p.Name()
		if err != nil {
			// Process exited — remove from cache and skip.
			delete(pc.cache, pid)
			continue
		}

		cpuPct, err := p.Percent(0)
		if err != nil {
			cpuPct = 0
		}

		var rss uint64
		var memPct float32
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			rss = mi.RSS
			if pc.totalMem > 0 {
				memPct = float32(100.0 * float64(rss) / float64(pc.totalMem))
			}
		}

		infos = append(infos, ProcessInfo{
			PID:        pid,
			Name:       name,
			CPUPercent: cpuPct,
			MemPercent: memPct,
			MemRSS:     rss,
			ConnCount:  -1, // populated separately by connection collector
		})
	}

	// Prune cached processes that no longer exist.
	for pid := range pc.cache {
		if _, ok := live[pid]; !ok {
			delete(pc.cache, pid)
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].CPUPercent > infos[j].CPUPercent
	})

	return infos, nil
}
