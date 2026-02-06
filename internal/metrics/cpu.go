package metrics

import (
	"github.com/shirou/gopsutil/v3/cpu"
)

func collectCPU() (CPUMetrics, error) {
	perCore, err := cpu.Percent(0, true)
	if err != nil {
		return CPUMetrics{}, err
	}

	total, err := cpu.Percent(0, false)
	if err != nil {
		return CPUMetrics{}, err
	}

	var totalPct float64
	if len(total) > 0 {
		totalPct = total[0]
	}

	return CPUMetrics{
		PerCore: perCore,
		Total:   totalPct,
	}, nil
}
