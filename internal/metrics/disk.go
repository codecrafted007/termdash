package metrics

import (
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
)

func collectDisks() ([]DiskMetrics, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var disks []DiskMetrics
	seen := make(map[string]bool)

	for _, p := range partitions {
		// Skip pseudo-filesystems.
		if strings.HasPrefix(p.Fstype, "devfs") ||
			strings.HasPrefix(p.Fstype, "tmpfs") ||
			strings.HasPrefix(p.Fstype, "proc") ||
			strings.HasPrefix(p.Fstype, "sysfs") {
			continue
		}

		// Deduplicate by mount point.
		if seen[p.Mountpoint] {
			continue
		}
		seen[p.Mountpoint] = true

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		if usage.Total == 0 {
			continue
		}

		disks = append(disks, DiskMetrics{
			MountPoint: p.Mountpoint,
			Device:     p.Device,
			Total:      usage.Total,
			Used:       usage.Used,
			Percent:    usage.UsedPercent,
		})
	}

	return disks, nil
}
