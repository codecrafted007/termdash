package panels

import (
	"fmt"
	"time"

	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderHeader renders the hostname, OS, uptime and current time.
// If recording is true, a red "REC●" indicator is shown after the title.
// If statusMsg is non-empty, it is appended after the time field.
func RenderHeader(snap metrics.Snapshot, recording bool, statusMsg string) string {
	title := styles.HeaderTitleStyle.Render("⣿ termdash")

	if recording {
		rec := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorDanger).Render(" REC●")
		title += rec
	}

	info := lipgloss.JoinHorizontal(lipgloss.Center,
		styles.LabelStyle.Render("Host: ")+styles.ValueStyle.Render(snap.Hostname),
		"  ",
		styles.LabelStyle.Render("OS: ")+styles.ValueStyle.Render(snap.OS),
		"  ",
		styles.LabelStyle.Render("Uptime: ")+styles.ValueStyle.Render(formatUptime(snap.Uptime)),
		"  ",
		styles.LabelStyle.Render("Time: ")+styles.ValueStyle.Render(snap.Timestamp.Format("15:04:05")),
	)

	parts := []string{title, "  ", info}
	if statusMsg != "" {
		status := lipgloss.NewStyle().Foreground(styles.ColorSuccess).Render("  " + statusMsg)
		parts = append(parts, status)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}
