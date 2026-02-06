package panels

import (
	"fmt"
	"strings"

	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderSummary renders 3 compact lines: CPU bar + per-core inline, MEM bar, Swap bar.
func RenderSummary(snap metrics.Snapshot, width int) string {
	if width < 20 {
		width = 20
	}

	// CPU line: CPU [████░░░░░░] 32%  0:45%  1:28%  2:35%
	barWidth := 20
	if width < 60 {
		barWidth = 10
	}

	cpuBar := summaryBar(snap.CPU.Total, barWidth, styles.ColorCPU)
	cpuLine := fmt.Sprintf("CPU %s %4.0f%%", cpuBar, snap.CPU.Total)

	// Add per-core inline if there's room
	coreStr := ""
	for i, pct := range snap.CPU.PerCore {
		part := fmt.Sprintf("  %d:%2.0f%%", i, pct)
		if len(cpuLine)+len(coreStr)+len(part) > width-2 {
			break
		}
		coreStr += part
	}
	cpuLine += coreStr

	// MEM line: Mem [██████░░░░]  6.2/16.0 GiB (54%)
	memBar := summaryBar(snap.Memory.RAMPercent, barWidth, styles.ColorRAM)
	memLine := fmt.Sprintf("Mem %s  %s / %s (%2.0f%%)",
		memBar,
		metrics.FormatBytes(snap.Memory.UsedRAM),
		metrics.FormatBytes(snap.Memory.TotalRAM),
		snap.Memory.RAMPercent,
	)

	// Swap line: Swp [░░░░░░░░░░]  0.0/2.0 GiB (0%)
	swapBar := summaryBar(snap.Memory.SwapPercent, barWidth, styles.ColorSwap)
	swapLine := fmt.Sprintf("Swp %s  %s / %s (%2.0f%%)",
		swapBar,
		metrics.FormatBytes(snap.Memory.UsedSwap),
		metrics.FormatBytes(snap.Memory.TotalSwap),
		snap.Memory.SwapPercent,
	)

	cpuStyled := styles.LabelStyle.Render(cpuLine)
	memStyled := styles.LabelStyle.Render(memLine)
	swapStyled := styles.LabelStyle.Render(swapLine)

	return lipgloss.JoinVertical(lipgloss.Left, cpuStyled, memStyled, swapStyled)
}

func summaryBar(pct float64, barWidth int, color lipgloss.Color) string {
	if barWidth < 1 {
		barWidth = 1
	}
	filled := int(pct / 100.0 * float64(barWidth))
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)

	return "[" + filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty)) + "]"
}
