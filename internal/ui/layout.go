package ui

import (
	"strings"

	"github.com/brajesh/termdash/pkg/dsl"
	"github.com/charmbracelet/lipgloss"
)

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

// WidgetRenderer is a function that renders a widget given the allocated width.
type WidgetRenderer func(width int) string

// RenderConfigLayout arranges the dashboard using the parsed layout config.
// widgets maps widget name → render function called with the actual column width.
// header, queryBar, and help are chrome elements rendered outside the config grid.
func RenderConfigLayout(layout *dsl.LayoutBlock, width, height int, widgets map[string]WidgetRenderer, header, queryBar, help string) string {
	if width < 40 {
		width = 40
	}

	headerRendered := lipgloss.NewStyle().Width(width).Render(header)
	helpRendered := lipgloss.NewStyle().Width(width).Render(help)

	// Calculate available height for the grid.
	// header ~1 line, help ~1 line, queryBar ~1 line if present.
	chromeHeight := lipgloss.Height(headerRendered) + lipgloss.Height(helpRendered)
	if queryBar != "" {
		chromeHeight += lipgloss.Height(queryBar)
	}
	gridHeight := height - chromeHeight
	if gridHeight < 1 {
		gridHeight = 1
	}

	// Sum row weights.
	totalWeight := 0
	for _, row := range layout.Rows {
		w := row.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}
	if totalWeight == 0 {
		totalWeight = 1
	}

	// Render each row.
	rowStrings := make([]string, 0, len(layout.Rows))
	heightUsed := 0
	for i, row := range layout.Rows {
		w := row.Weight
		if w <= 0 {
			w = 1
		}
		var rowHeight int
		if i == len(layout.Rows)-1 {
			// Last row gets remaining height to avoid rounding gaps.
			rowHeight = gridHeight - heightUsed
		} else {
			rowHeight = gridHeight * w / totalWeight
		}
		if rowHeight < 1 {
			rowHeight = 1
		}
		heightUsed += rowHeight

		rowStr := renderConfigRow(row, width, rowHeight, widgets)
		rowStrings = append(rowStrings, rowStr)
	}

	parts := []string{headerRendered}
	parts = append(parts, rowStrings...)
	if queryBar != "" {
		parts = append(parts, queryBar)
	}
	parts = append(parts, helpRendered)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderConfigRow renders a single row with columns distributed by weight.
func renderConfigRow(row dsl.RowBlock, width, height int, widgets map[string]WidgetRenderer) string {
	if len(row.Cols) == 0 {
		return strings.Repeat("\n", height-1)
	}

	// Sum col weights.
	totalWeight := 0
	for _, col := range row.Cols {
		w := col.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w
	}
	if totalWeight == 0 {
		totalWeight = 1
	}

	cols := make([]string, 0, len(row.Cols))
	widthUsed := 0
	for i, col := range row.Cols {
		w := col.Weight
		if w <= 0 {
			w = 1
		}
		var colWidth int
		if i == len(row.Cols)-1 {
			colWidth = width - widthUsed
		} else {
			colWidth = width * w / totalWeight
		}
		if colWidth < 1 {
			colWidth = 1
		}
		widthUsed += colWidth

		content := ""
		if fn, ok := widgets[col.Widget]; ok {
			content = fn(colWidth)
		}
		rendered := lipgloss.NewStyle().
			Width(colWidth).
			MaxWidth(colWidth).
			Height(height).
			MaxHeight(height).
			Render(content)
		cols = append(cols, rendered)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cols...)
}
