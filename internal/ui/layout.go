package ui

import "github.com/charmbracelet/lipgloss"

// RenderDashboardLayout arranges the dashboard view as a vertical stack:
// header, summary, process table, queryBar (if active), netTop, help.
func RenderDashboardLayout(width, height int, header, summary, processTable, queryBar, netTop, help string) string {
	if width < 40 {
		width = 40
	}

	headerRendered := lipgloss.NewStyle().Width(width).Render(header)
	summaryRendered := lipgloss.NewStyle().Width(width).Render(summary)
	helpRendered := lipgloss.NewStyle().Width(width).Render(help)

	parts := []string{headerRendered, summaryRendered, processTable}
	if queryBar != "" {
		parts = append(parts, queryBar)
	}
	if netTop != "" {
		parts = append(parts, netTop)
	}
	parts = append(parts, helpRendered)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// RenderDetailLayout arranges the detail view as a full-screen vertical stack.
func RenderDetailLayout(width, height int, detailView, help string) string {
	if width < 40 {
		width = 40
	}

	helpRendered := lipgloss.NewStyle().Width(width).Render(help)

	return lipgloss.JoinVertical(lipgloss.Left, detailView, helpRendered)
}

// RenderLayout is the legacy layout function kept for backward compatibility.
// It delegates to the old 4-panel layout. Old panel renderers still reference it.
func RenderLayout(width int, header, cpuPanel, memPanel, diskPanel, netPanel, processPanel, help string) string {
	if width < 40 {
		width = 40
	}

	headerRendered := lipgloss.NewStyle().Width(width).Render(header)
	helpRendered := lipgloss.NewStyle().Width(width).Render(help)

	var body string
	if width >= 80 {
		leftCol := lipgloss.JoinVertical(lipgloss.Left, cpuPanel, memPanel)
		rightCol := lipgloss.JoinVertical(lipgloss.Left, diskPanel, netPanel)
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol)
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, cpuPanel, memPanel, diskPanel, netPanel)
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerRendered, body, processPanel, helpRendered)
}
