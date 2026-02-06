package panels

import (
	"fmt"
	"strings"

	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderProcesses renders a table of the top processes by CPU usage.
func RenderProcesses(snap metrics.Snapshot, width int) string {
	title := styles.SectionTitleStyle.Render("Processes")

	innerWidth := width - styles.PanelBorderOverhead

	// Column widths: PID(7) + Name(variable) + CPU%(8) + MEM%(8) + User(12) + separators(8)
	const (
		pidW  = 7
		cpuW  = 8
		memW  = 8
		userW = 12
		gaps  = 8 // spaces between columns
	)
	nameW := innerWidth - pidW - cpuW - memW - userW - gaps
	if nameW < 8 {
		nameW = 8
	}

	// Header
	header := fmt.Sprintf(" %-*s  %-*s  %*s  %*s  %-*s",
		pidW, "PID",
		nameW, "NAME",
		cpuW, "CPU%",
		memW, "MEM%",
		userW, "USER",
	)
	headerLine := styles.LabelStyle.Render(header)
	separator := styles.LabelStyle.Render(" " + strings.Repeat("─", innerWidth-2))

	var rows strings.Builder
	for _, p := range snap.Processes {
		name := p.Name
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}
		user := p.User
		if len(user) > userW {
			user = user[:userW-1] + "…"
		}

		row := fmt.Sprintf(" %-*d  %-*s  %*.1f  %*.1f  %-*s",
			pidW, p.PID,
			nameW, name,
			cpuW, p.CPUPercent,
			memW, float64(p.MemPercent),
			userW, user,
		)
		rows.WriteString(row + "\n")
	}

	if len(snap.Processes) == 0 {
		rows.WriteString(styles.LabelStyle.Render(" No process data available"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, title, headerLine, separator, rows.String())
	return styles.PanelWithWidth(width).Render(content)
}
