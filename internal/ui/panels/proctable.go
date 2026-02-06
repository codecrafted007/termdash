package panels

import (
	"fmt"
	"strings"

	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// SortColumn mirrors the ui.SortColumn type for the panel renderer.
type SortColumn int

const (
	SortByCPU  SortColumn = iota
	SortByMem
	SortByPID
	SortByName
	SortByConns
)

// TableRow represents a single row in the grouped process table.
type TableRow struct {
	PID        string  // "(38)" for group, "1234" for process
	Name       string
	User       string
	CPUPercent float64
	MemPercent float64
	MemRSS     uint64
	ConnCount  int
	IsGroup    bool
	Expanded   bool
	Depth      int // 0 = top level, 1 = child of expanded group
}

// RenderProcessTable renders a scrollable, selectable process table.
func RenderProcessTable(
	processes []metrics.ProcessInfo,
	connCounts map[int32]int,
	cursor, scrollOffset, visibleRows int,
	sortCol SortColumn, sortAsc bool,
	width int,
) string {
	if width < 40 {
		width = 40
	}
	innerWidth := width - 2 // leave 1 char margin each side

	// Column widths
	const (
		pidW  = 7
		cpuW  = 6
		memW  = 6
		rssW  = 8
		connW = 5
		userW = 10
		gaps  = 9 // spaces between columns + leading space
	)
	nameW := innerWidth - pidW - cpuW - memW - rssW - connW - userW - gaps
	if nameW < 6 {
		nameW = 6
	}

	// Header with sort indicators
	header := fmt.Sprintf(" %-*s %-*s %-*s %*s %*s %*s %*s",
		pidW, sortHeader("PID", SortByPID, sortCol, sortAsc),
		nameW, sortHeader("NAME", SortByName, sortCol, sortAsc),
		userW, "USER",
		cpuW, sortHeader("CPU%", SortByCPU, sortCol, sortAsc),
		memW, sortHeader("MEM%", SortByMem, sortCol, sortAsc),
		rssW, "RSS",
		connW, sortHeader("CONN", SortByConns, sortCol, sortAsc),
	)
	headerLine := styles.LabelStyle.Render(header)
	separator := styles.LabelStyle.Render(" " + strings.Repeat("─", innerWidth-2))

	// Scroll indicators
	hasAbove := scrollOffset > 0
	hasBelow := scrollOffset+visibleRows < len(processes)

	var rows strings.Builder

	if hasAbove {
		rows.WriteString(styles.LabelStyle.Render(fmt.Sprintf(" %*s", innerWidth/2, "▲ more")) + "\n")
		visibleRows-- // use one row for indicator
	}

	if hasBelow {
		visibleRows-- // reserve one row for bottom indicator
	}

	end := scrollOffset + visibleRows
	if end > len(processes) {
		end = len(processes)
	}

	for i := scrollOffset; i < end; i++ {
		p := processes[i]
		name := p.Name
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}
		user := p.User
		if len(user) > userW {
			user = user[:userW-1] + "…"
		}

		rss := metrics.FormatBytes(p.MemRSS)
		if len(rss) > rssW {
			rss = rss[:rssW]
		}

		conn := "-"
		if c, ok := connCounts[p.PID]; ok {
			conn = fmt.Sprintf("%d", c)
		} else if p.ConnCount >= 0 {
			conn = fmt.Sprintf("%d", p.ConnCount)
		}

		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		row := fmt.Sprintf("%s%-*d %-*s %-*s %*.1f %*.1f %*s %*s",
			prefix,
			pidW, p.PID,
			nameW, name,
			userW, user,
			cpuW, p.CPUPercent,
			memW, float64(p.MemPercent),
			rssW, rss,
			connW, conn,
		)

		if i == cursor {
			row = styles.CursorRowStyle.Render(row)
		}

		rows.WriteString(row + "\n")
	}

	if hasBelow {
		rows.WriteString(styles.LabelStyle.Render(fmt.Sprintf(" %*s", innerWidth/2, "▼ more")))
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerLine, separator, rows.String())
}

func sortHeader(label string, col, activeCol SortColumn, asc bool) string {
	if col == activeCol {
		arrow := "▼"
		if asc {
			arrow = "▲"
		}
		return styles.SortIndicatorStyle.Render(label + arrow)
	}
	return label
}

// RenderGroupedProcessTable renders a scrollable process table with grouping support.
func RenderGroupedProcessTable(
	rows []TableRow,
	cursor, scrollOffset, visibleRows int,
	sortCol SortColumn, sortAsc bool,
	width int,
) string {
	if width < 40 {
		width = 40
	}
	innerWidth := width - 2

	// Column widths
	const (
		pidW  = 7
		cpuW  = 6
		memW  = 6
		rssW  = 8
		connW = 5
		userW = 10
		gaps  = 9
	)
	nameW := innerWidth - pidW - cpuW - memW - rssW - connW - userW - gaps
	if nameW < 6 {
		nameW = 6
	}

	// Header with sort indicators
	header := fmt.Sprintf(" %-*s %-*s %-*s %*s %*s %*s %*s",
		pidW, sortHeader("PID", SortByPID, sortCol, sortAsc),
		nameW, sortHeader("NAME", SortByName, sortCol, sortAsc),
		userW, "USER",
		cpuW, sortHeader("CPU%", SortByCPU, sortCol, sortAsc),
		memW, sortHeader("MEM%", SortByMem, sortCol, sortAsc),
		rssW, "RSS",
		connW, sortHeader("CONN", SortByConns, sortCol, sortAsc),
	)
	headerLine := styles.LabelStyle.Render(header)
	separator := styles.LabelStyle.Render(" " + strings.Repeat("─", innerWidth-2))

	// Scroll indicators
	hasAbove := scrollOffset > 0
	hasBelow := scrollOffset+visibleRows < len(rows)

	var rowsBuilder strings.Builder

	if hasAbove {
		rowsBuilder.WriteString(styles.LabelStyle.Render(fmt.Sprintf(" %*s", innerWidth/2, "▲ more")) + "\n")
		visibleRows--
	}

	if hasBelow {
		visibleRows--
	}

	end := scrollOffset + visibleRows
	if end > len(rows) {
		end = len(rows)
	}

	for i := scrollOffset; i < end; i++ {
		r := rows[i]

		// Determine prefix based on group state and cursor
		var prefix string
		if i == cursor {
			prefix = "> "
		} else {
			prefix = "  "
		}

		// Group/expand indicator
		indicator := " "
		if r.IsGroup {
			if r.Expanded {
				indicator = "▼"
			} else {
				indicator = "▶"
			}
		}

		// Indented name for children
		name := r.Name
		if r.Depth > 0 {
			// Tree character for child rows
			name = "├─" + name
		}
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}

		// PID column (with indicator for groups)
		pidStr := r.PID
		if r.IsGroup {
			pidStr = indicator + r.PID
		} else if r.Depth > 0 {
			pidStr = "  " + r.PID // indent child PIDs
		}
		if len(pidStr) > pidW {
			pidStr = pidStr[:pidW]
		}

		user := r.User
		if len(user) > userW {
			user = user[:userW-1] + "…"
		}

		rss := metrics.FormatBytes(r.MemRSS)
		if len(rss) > rssW {
			rss = rss[:rssW]
		}

		conn := "-"
		if r.ConnCount >= 0 {
			conn = fmt.Sprintf("%d", r.ConnCount)
		}

		row := fmt.Sprintf("%s%-*s %-*s %-*s %*.1f %*.1f %*s %*s",
			prefix,
			pidW, pidStr,
			nameW, name,
			userW, user,
			cpuW, r.CPUPercent,
			memW, r.MemPercent,
			rssW, rss,
			connW, conn,
		)

		if i == cursor {
			row = styles.CursorRowStyle.Render(row)
		}

		rowsBuilder.WriteString(row + "\n")
	}

	if hasBelow {
		rowsBuilder.WriteString(styles.LabelStyle.Render(fmt.Sprintf(" %*s", innerWidth/2, "▼ more")))
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerLine, separator, rowsBuilder.String())
}
