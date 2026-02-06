package metrics

import "fmt"

// FormatBytes formats a byte count into a human-readable string (e.g., "1.5 GiB").
func FormatBytes(b uint64) string {
	const (
		KiB = 1024
		MiB = KiB * 1024
		GiB = MiB * 1024
		TiB = GiB * 1024
	)

	switch {
	case b >= TiB:
		return fmt.Sprintf("%.1f TiB", float64(b)/float64(TiB))
	case b >= GiB:
		return fmt.Sprintf("%.1f GiB", float64(b)/float64(GiB))
	case b >= MiB:
		return fmt.Sprintf("%.1f MiB", float64(b)/float64(MiB))
	case b >= KiB:
		return fmt.Sprintf("%.1f KiB", float64(b)/float64(KiB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// FormatRate formats a bytes-per-second rate into a human-readable string (e.g., "1.5 MiB/s").
func FormatRate(bytesPerSec float64) string {
	const (
		KiB = 1024.0
		MiB = KiB * 1024
		GiB = MiB * 1024
	)

	switch {
	case bytesPerSec >= GiB:
		return fmt.Sprintf("%.1f GiB/s", bytesPerSec/GiB)
	case bytesPerSec >= MiB:
		return fmt.Sprintf("%.1f MiB/s", bytesPerSec/MiB)
	case bytesPerSec >= KiB:
		return fmt.Sprintf("%.1f KiB/s", bytesPerSec/KiB)
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
}
