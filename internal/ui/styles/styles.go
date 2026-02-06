package styles

import "github.com/charmbracelet/lipgloss"

// Color palette.
var (
	ColorPrimary   = lipgloss.Color("#7D56F4") // purple
	ColorSecondary = lipgloss.Color("#6C63FF") // indigo
	ColorSuccess   = lipgloss.Color("#04B575") // green
	ColorWarning   = lipgloss.Color("#FFBE0B") // amber
	ColorDanger    = lipgloss.Color("#FF6B6B") // red
	ColorMuted     = lipgloss.Color("#626262") // dim gray
	ColorText      = lipgloss.Color("#FAFAFA") // near-white
	ColorSubtle    = lipgloss.Color("#A0A0A0") // light gray
	ColorCPU       = lipgloss.Color("#7D56F4") // purple
	ColorRAM       = lipgloss.Color("#04B575") // green
	ColorSwap      = lipgloss.Color("#FFBE0B") // amber
	ColorDisk      = lipgloss.Color("#6C63FF") // indigo
	ColorNetSend   = lipgloss.Color("#FF6B6B") // red
	ColorNetRecv   = lipgloss.Color("#04B575") // green
)

// UsageColor returns a color based on usage percentage thresholds.
func UsageColor(pct float64) lipgloss.Color {
	switch {
	case pct >= 90:
		return ColorDanger
	case pct >= 70:
		return ColorWarning
	default:
		return ColorSuccess
	}
}

// PanelBorderOverhead is the horizontal space consumed by the panel border (2)
// and padding (2): 1 left border + 1 left pad + 1 right pad + 1 right border.
const PanelBorderOverhead = 4

// Panel border style.
var PanelStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorMuted).
	Padding(0, 1)

// PanelWithWidth returns PanelStyle with a content width set so the total
// outer width (including border + padding) equals outerWidth.
func PanelWithWidth(outerWidth int) lipgloss.Style {
	return PanelStyle.Width(outerWidth - PanelBorderOverhead)
}

// Header title style.
var HeaderTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorPrimary)

// Section title style used inside panels.
var SectionTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorText).
	MarginBottom(1)

// Label style for metric names.
var LabelStyle = lipgloss.NewStyle().
	Foreground(ColorSubtle)

// Value style for metric values.
var ValueStyle = lipgloss.NewStyle().
	Foreground(ColorText).
	Bold(true)

// Help key style.
var HelpKeyStyle = lipgloss.NewStyle().
	Foreground(ColorPrimary).
	Bold(true)

// Help description style.
var HelpDescStyle = lipgloss.NewStyle().
	Foreground(ColorMuted)

// ColorConn is the color for connection count column.
var ColorConn = lipgloss.Color("#FF8C00") // orange

// CursorRowStyle highlights the selected process row.
var CursorRowStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#3A3A5C")).
	Foreground(ColorText)

// SortIndicatorStyle for sort arrows on column headers.
var SortIndicatorStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorPrimary)

// DetailHeaderStyle for the detail view title bar.
var DetailHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorText).
	Background(lipgloss.Color("#3A3A5C")).
	Padding(0, 1)
