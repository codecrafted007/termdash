package panels

import (
	"fmt"
	"strings"

	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderProcessDetail renders the full detail view for a single process.
func RenderProcessDetail(detail *metrics.ProcessDetail, cpuHistory, memHistory []float64, width, height int) string {
	if width < 40 {
		width = 40
	}
	innerWidth := width - 4

	if detail == nil {
		return lipgloss.NewStyle().Width(width).Render(
			"\n" + styles.LabelStyle.Render("  Loading process details..."),
		)
	}

	var sections []string

	// Title bar
	title := styles.DetailHeaderStyle.Width(innerWidth).Render(
		fmt.Sprintf("Process: %s (PID %d)  ── [Esc] back", detail.Name, detail.PID),
	)
	sections = append(sections, title)

	// Info lines
	fdsStr := fmt.Sprintf("%d", detail.NumFDs)
	if detail.NumFDs < 0 {
		fdsStr = "N/A"
	}
	connStr := fmt.Sprintf("%d", detail.ConnCount)

	info1 := fmt.Sprintf("User: %-12s  Status: %-4s  Threads: %-5d  FDs: %s",
		detail.User, detail.Status, detail.NumThreads, fdsStr)
	info2 := fmt.Sprintf("Parent: %-7d  Nice: %-4d  Conns: %-5s  Created: %s",
		detail.ParentPID, detail.Nice, connStr,
		detail.CreateTime.Format("2006-01-02 15:04:05"))

	sections = append(sections,
		styles.LabelStyle.Render(info1),
		styles.LabelStyle.Render(info2),
	)

	// Cmdline
	cmdline := detail.Cmdline
	if cmdline == "" {
		cmdline = detail.Exe
	}
	if len(cmdline) > innerWidth-6 {
		cmdline = cmdline[:innerWidth-9] + "..."
	}
	sections = append(sections,
		styles.LabelStyle.Render("Cmd: ")+styles.ValueStyle.Render(cmdline),
	)

	// Separator
	sections = append(sections, styles.LabelStyle.Render(strings.Repeat("─", innerWidth)))

	// Sparkline charts side by side
	chartWidth := (innerWidth - 3) / 2 // 3 for " │ " separator
	if chartWidth < 10 {
		chartWidth = 10
	}
	chartHeight := 3

	leftChart := renderDetailSparkline("CPU History", cpuHistory, chartWidth, chartHeight, styles.ColorCPU, 100)
	rightChart := renderDetailSparkline("Memory History", memHistory, chartWidth, chartHeight, styles.ColorRAM, 100)

	cpuCurrent := fmt.Sprintf("Current: %.1f%%", detail.CPUPercent)
	memCurrent := fmt.Sprintf("RSS: %s  VMS: %s",
		metrics.FormatBytes(detail.MemRSS),
		metrics.FormatBytes(detail.MemVMS))

	leftPanel := lipgloss.JoinVertical(lipgloss.Left, leftChart, styles.ValueStyle.Render(cpuCurrent))
	rightPanel := lipgloss.JoinVertical(lipgloss.Left, rightChart, styles.ValueStyle.Render(memCurrent))

	charts := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " │ ", rightPanel)
	sections = append(sections, charts)

	// Separator
	sections = append(sections, styles.LabelStyle.Render(strings.Repeat("─", innerWidth)))

	// Environment
	sections = append(sections, styles.SectionTitleStyle.Render("Environment:"))
	if len(detail.Environ) == 0 {
		sections = append(sections, styles.LabelStyle.Render("  (not available — may require elevated privileges)"))
	} else {
		// Calculate how many env vars we can show
		envLines := height - len(sections) - 3 // leave room for help + padding
		if envLines < 1 {
			envLines = 1
		}
		if envLines > len(detail.Environ) {
			envLines = len(detail.Environ)
		}
		for _, env := range detail.Environ[:envLines] {
			if len(env) > innerWidth {
				env = env[:innerWidth-3] + "..."
			}
			sections = append(sections, styles.LabelStyle.Render("  "+env))
		}
		if envLines < len(detail.Environ) {
			sections = append(sections,
				styles.LabelStyle.Render(fmt.Sprintf("  ... and %d more", len(detail.Environ)-envLines)))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func renderDetailSparkline(label string, data []float64, width, height int, color lipgloss.Color, maxVal float64) string {
	header := styles.LabelStyle.Render(label)
	if len(data) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, header,
			styles.LabelStyle.Render(strings.Repeat("░", width)))
	}

	s := sparkline.New(width, height,
		sparkline.WithMaxValue(maxVal),
		sparkline.WithStyle(lipgloss.NewStyle().Foreground(color)),
	)
	s.PushAll(data)
	s.Draw()

	return lipgloss.JoinVertical(lipgloss.Left, header, s.View())
}
