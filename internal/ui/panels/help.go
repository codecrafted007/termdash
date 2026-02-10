package panels

import (
	"github.com/brajesh/termdash/internal/ui/styles"
	"github.com/charmbracelet/lipgloss"
)

// RenderHelp renders the keybindings footer.
// inDetail controls which key set is shown.
func RenderHelp(showHelp bool, inDetail bool) string {
	if !showHelp {
		if inDetail {
			return styles.HelpDescStyle.Render(" Esc:back  e:export  ?:help  q:quit")
		}
		return styles.HelpDescStyle.Render(" j/k:move  /:search  Q:query  s:sort  p:group  t:replay  e:export  ?:help  q:quit")
	}

	var keys []struct{ key, desc string }
	if inDetail {
		keys = []struct{ key, desc string }{
			{"Esc", "Back to dashboard"},
			{"e", "Export JSON snapshot"},
			{"E", "Toggle CSV recording"},
			{"?", "Toggle help"},
			{"q", "Quit"},
			{"Ctrl+C", "Force quit"},
		}
	} else {
		keys = []struct{ key, desc string }{
			{"j/k", "Move cursor"},
			{"/", "Search by name"},
			{"Q", "Query filter (DSL)"},
			{"Enter", "Process detail"},
			{"s", "Cycle sort column"},
			{"S", "Toggle sort direction"},
			{"g/G", "Top/bottom"},
			{"p", "Toggle process grouping"},
			{"t", "Replay history"},
			{"e", "Export JSON snapshot"},
			{"E", "Toggle CSV recording"},
			{"?", "Toggle help"},
			{"q", "Quit"},
		}
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts,
			styles.HelpKeyStyle.Render(k.key)+" "+styles.HelpDescStyle.Render(k.desc),
		)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, " ",
		lipgloss.JoinHorizontal(lipgloss.Center, joinWith(parts, "  ")...),
	)
}

func joinWith(parts []string, sep string) []string {
	if len(parts) == 0 {
		return nil
	}
	result := []string{parts[0]}
	for _, p := range parts[1:] {
		result = append(result, sep, p)
	}
	return result
}
