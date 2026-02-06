package panels

import (
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// QueryType distinguishes simple search from field queries.
// Duplicated here to avoid import cycle with ui package.
type QueryType int

const (
	QuerySearch QueryType = iota // "/" simple search
	QueryFilter                  // "Q" field-based DSL
)

var (
	queryBarStyle = lipgloss.NewStyle().
			Foreground(styles.ColorText).
			Background(lipgloss.Color("#2A2A3C")).
			Padding(0, 1)

	queryPromptStyle = lipgloss.NewStyle().
				Foreground(styles.ColorPrimary).
				Bold(true)

	queryErrorStyle = lipgloss.NewStyle().
			Foreground(styles.ColorDanger).
			Italic(true)

	queryFilteredStyle = lipgloss.NewStyle().
				Foreground(styles.ColorSuccess).
				Italic(true)
)

// RenderQueryBar renders the query input bar at the bottom of the screen.
func RenderQueryBar(queryType QueryType, input textinput.Model, errorMsg string, filterLocked bool, width int) string {
	var prompt string
	if queryType == QuerySearch {
		prompt = "/"
	} else {
		prompt = "Query: "
	}

	promptRendered := queryPromptStyle.Render(prompt)

	var content string
	if filterLocked {
		// Show locked filter status
		content = promptRendered + input.Value() + " " + queryFilteredStyle.Render("[locked - Esc to clear]")
	} else {
		// Show active input
		content = promptRendered + input.View()
		if errorMsg != "" {
			content += "  " + queryErrorStyle.Render(errorMsg)
		}
	}

	return queryBarStyle.Width(width - 2).Render(content)
}
