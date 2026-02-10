package dsl

// DefaultConfig returns the config equivalent of the current hardcoded layout:
// row 1 (weight 1): summary + nettop side by side
// row 2 (weight 3): procs full-width
// refresh = 2s
func DefaultConfig() *Config {
	return &Config{
		Global: &GlobalBlock{
			Refresh: "2s",
		},
		Layout: &LayoutBlock{
			Rows: []RowBlock{
				{
					Weight: 1,
					Cols: []ColBlock{
						{Weight: 3, Widget: "summary"},
						{Weight: 2, Widget: "nettop"},
					},
				},
				{
					Weight: 3,
					Cols: []ColBlock{
						{Weight: 1, Widget: "procs"},
					},
				},
			},
		},
	}
}

// KnownWidgets is the set of widget names supported in milestone 1.
var KnownWidgets = map[string]bool{
	"summary": true,
	"procs":   true,
	"nettop":  true,
	"cpu":     true,
	"memory":  true,
}
