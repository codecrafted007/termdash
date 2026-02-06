package panels

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderNetwork renders network I/O rates with sparklines.
func RenderNetwork(snap metrics.Snapshot, sendHistory, recvHistory []float64, width int) string {
	title := styles.SectionTitleStyle.Render("Network")

	sendLine := fmt.Sprintf(" %s %s",
		styles.LabelStyle.Render("Send: "),
		styles.ValueStyle.Foreground(styles.ColorNetSend).Render(metrics.FormatRate(snap.Network.SendRate)),
	)
	recvLine := fmt.Sprintf(" %s %s",
		styles.LabelStyle.Render("Recv: "),
		styles.ValueStyle.Foreground(styles.ColorNetRecv).Render(metrics.FormatRate(snap.Network.RecvRate)),
	)

	totalLine := fmt.Sprintf(" %s %s  %s %s",
		styles.LabelStyle.Render("Total Sent:"),
		metrics.FormatBytes(snap.Network.BytesSent),
		styles.LabelStyle.Render("Total Recv:"),
		metrics.FormatBytes(snap.Network.BytesRecv),
	)

	innerWidth := width - styles.PanelBorderOverhead
	chartWidth := max(innerWidth-2, 10)

	sendSpark := ""
	if len(sendHistory) > 0 {
		s := sparkline.New(chartWidth, 2,
			sparkline.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorNetSend)),
		)
		s.PushAll(sendHistory)
		s.Draw()
		sendSpark = " " + styles.LabelStyle.Render("Send") + "\n" + s.View()
	}

	recvSpark := ""
	if len(recvHistory) > 0 {
		r := sparkline.New(chartWidth, 2,
			sparkline.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorNetRecv)),
		)
		r.PushAll(recvHistory)
		r.Draw()
		recvSpark = " " + styles.LabelStyle.Render("Recv") + "\n" + r.View()
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		sendLine,
		recvLine,
		"",
		totalLine,
		"",
		sendSpark,
		recvSpark,
	)

	return styles.PanelWithWidth(width).Render(content)
}
