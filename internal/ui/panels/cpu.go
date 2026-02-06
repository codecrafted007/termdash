package panels

import (
	"fmt"
	"strings"

	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderCPU renders per-core CPU bars and a sparkline for total CPU history.
func RenderCPU(snap metrics.Snapshot, cpuHistory []float64, width int) string {
	title := styles.SectionTitleStyle.Render("CPU")

	innerWidth := width - styles.PanelBorderOverhead

	// Per-core bars.
	var cores strings.Builder
	for i, pct := range snap.CPU.PerCore {
		label := fmt.Sprintf("Core %-2d", i)
		bar := renderBar(pct, innerWidth-14, styles.ColorCPU)
		cores.WriteString(fmt.Sprintf(" %s %s %5.1f%%\n", styles.LabelStyle.Render(label), bar, pct))
	}

	totalLabel := fmt.Sprintf(" %s %s %5.1f%%",
		styles.LabelStyle.Render("Total  "),
		renderBar(snap.CPU.Total, innerWidth-14, lipgloss.Color(string(styles.UsageColor(snap.CPU.Total)))),
		snap.CPU.Total,
	)

	// Sparkline for CPU total history.
	sparkStr := ""
	if len(cpuHistory) > 0 {
		chartWidth := max(innerWidth-2, 10)
		s := sparkline.New(chartWidth, 3,
			sparkline.WithMaxValue(100),
			sparkline.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorCPU)),
		)
		s.PushAll(cpuHistory)
		s.Draw()
		sparkStr = "\n" + styles.LabelStyle.Render(" History ") + "\n" + s.View()
	}

	return styles.PanelWithWidth(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, cores.String(), totalLabel, sparkStr),
	)
}

// renderBar creates a simple horizontal bar of a given width.
func renderBar(pct float64, maxWidth int, color lipgloss.Color) string {
	maxWidth = max(maxWidth, 1)
	filled := min(int(pct/100.0*float64(maxWidth)), maxWidth)
	if filled < 0 {
		filled = 0
	}
	empty := maxWidth - filled

	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(styles.ColorMuted)

	return filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))
}
