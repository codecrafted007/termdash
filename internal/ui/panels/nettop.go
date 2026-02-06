package panels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderNetTop renders a compact 2-line panel with aggregate network rates
// and top 5 processes by connection count.
func RenderNetTop(
	processes []metrics.ProcessInfo,
	connCounts map[int32]int,
	snap metrics.Snapshot,
	width int,
) string {
	sendRate := metrics.FormatRate(snap.Network.SendRate)
	recvRate := metrics.FormatRate(snap.Network.RecvRate)

	rateStr := fmt.Sprintf("Net: ▲%s ▼%s", sendRate, recvRate)

	// Top processes by connection count
	type procConn struct {
		name  string
		count int
	}
	var top []procConn
	seen := map[string]int{} // aggregate by name
	for _, p := range processes {
		c := 0
		if cc, ok := connCounts[p.PID]; ok {
			c = cc
		} else if p.ConnCount >= 0 {
			c = p.ConnCount
		}
		if c > 0 {
			seen[p.Name] += c
		}
	}
	for name, count := range seen {
		top = append(top, procConn{name: name, count: count})
	}
	sort.Slice(top, func(i, j int) bool {
		return top[i].count > top[j].count
	})
	if len(top) > 5 {
		top = top[:5]
	}

	var topParts []string
	for _, t := range top {
		topParts = append(topParts, fmt.Sprintf("%s(%d)", t.name, t.count))
	}

	topStr := ""
	if len(topParts) > 0 {
		topStr = " Top: " + strings.Join(topParts, " ")
	}

	// Truncate topStr if it would overflow
	if topStr != "" {
		sep := " │"
		maxTop := width - len(rateStr) - len(sep) - 2
		if maxTop > 0 && len(topStr) > maxTop {
			topStr = topStr[:maxTop-1] + "…"
		}
		if maxTop <= 0 {
			topStr = ""
		}
	}

	sendStyle := lipgloss.NewStyle().Foreground(styles.ColorNetSend)
	recvStyle := lipgloss.NewStyle().Foreground(styles.ColorNetRecv)
	connStyle := lipgloss.NewStyle().Foreground(styles.ColorConn)

	// Re-render with styles
	styledLine := fmt.Sprintf("Net: %s%s %s%s",
		sendStyle.Render("▲"), sendStyle.Render(sendRate),
		recvStyle.Render("▼"), recvStyle.Render(recvRate),
	)

	if topStr != "" {
		styledTop := connStyle.Render(topStr)
		styledLine += " │" + styledTop
	}

	return styles.LabelStyle.Render(styledLine)
}
