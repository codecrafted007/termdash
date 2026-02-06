package panels

import (
	"fmt"

	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderMemory renders RAM and swap usage bars.
func RenderMemory(snap metrics.Snapshot, width int) string {
	title := styles.SectionTitleStyle.Render("Memory")

	innerWidth := width - styles.PanelBorderOverhead
	barWidth := max(innerWidth-14, 1)

	ramBar := renderBar(snap.Memory.RAMPercent, barWidth, styles.ColorRAM)
	ramLine := fmt.Sprintf(" %s %s %5.1f%%",
		styles.LabelStyle.Render("RAM    "),
		ramBar,
		snap.Memory.RAMPercent,
	)
	ramDetail := fmt.Sprintf(" %s %s / %s",
		styles.LabelStyle.Render("       "),
		metrics.FormatBytes(snap.Memory.UsedRAM),
		metrics.FormatBytes(snap.Memory.TotalRAM),
	)

	swapBar := renderBar(snap.Memory.SwapPercent, barWidth, styles.ColorSwap)
	swapLine := fmt.Sprintf(" %s %s %5.1f%%",
		styles.LabelStyle.Render("Swap   "),
		swapBar,
		snap.Memory.SwapPercent,
	)
	swapDetail := fmt.Sprintf(" %s %s / %s",
		styles.LabelStyle.Render("       "),
		metrics.FormatBytes(snap.Memory.UsedSwap),
		metrics.FormatBytes(snap.Memory.TotalSwap),
	)

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		ramLine,
		ramDetail,
		"",
		swapLine,
		swapDetail,
	)

	return styles.PanelWithWidth(width).Render(content)
}
