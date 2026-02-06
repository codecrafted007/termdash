package panels

import (
	"fmt"
	"strings"

	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderDisk renders per-mount disk usage bars.
func RenderDisk(snap metrics.Snapshot, width int) string {
	title := styles.SectionTitleStyle.Render("Disk")

	innerWidth := width - styles.PanelBorderOverhead
	barWidth := max(innerWidth-14, 1)

	var lines strings.Builder
	for _, d := range snap.Disks {
		mount := d.MountPoint
		if len(mount) > 12 {
			mount = "..." + mount[len(mount)-9:]
		}
		label := fmt.Sprintf("%-12s", mount)
		color := styles.UsageColor(d.Percent)
		bar := renderBar(d.Percent, barWidth, color)
		lines.WriteString(fmt.Sprintf(" %s %s %5.1f%%\n",
			styles.LabelStyle.Render(label),
			bar,
			d.Percent,
		))
		lines.WriteString(fmt.Sprintf(" %s %s / %s\n",
			styles.LabelStyle.Render("            "),
			metrics.FormatBytes(d.Used),
			metrics.FormatBytes(d.Total),
		))
	}

	if len(snap.Disks) == 0 {
		lines.WriteString(styles.LabelStyle.Render(" No disks detected"))
	}

	return styles.PanelWithWidth(width).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, lines.String()),
	)
}
