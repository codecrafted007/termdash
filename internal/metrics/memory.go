package metrics

import (
	"github.com/shirou/gopsutil/v3/mem"
)

func collectMemory() (MemoryMetrics, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return MemoryMetrics{}, err
	}

	sw, err := mem.SwapMemory()
	if err != nil {
		return MemoryMetrics{}, err
	}

	return MemoryMetrics{
		TotalRAM:    vm.Total,
		UsedRAM:     vm.Used,
		RAMPercent:  vm.UsedPercent,
		TotalSwap:   sw.Total,
		UsedSwap:    sw.Used,
		SwapPercent: sw.UsedPercent,
	}, nil
}
