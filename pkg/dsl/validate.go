package dsl

import (
	"fmt"
	"time"
)

// Validate checks the config for semantic errors after HCL parsing.
func Validate(cfg *Config) error {
	if cfg.Global != nil {
		if cfg.Global.Refresh != "" {
			if _, err := time.ParseDuration(cfg.Global.Refresh); err != nil {
				return fmt.Errorf("global.refresh: invalid duration %q: %w", cfg.Global.Refresh, err)
			}
		}
		if cfg.Global.History != "" && cfg.Global.History != "0" {
			if _, err := time.ParseDuration(cfg.Global.History); err != nil {
				return fmt.Errorf("global.history: invalid duration %q: %w", cfg.Global.History, err)
			}
		}
		if cfg.Global.HistoryInterval != "" {
			d, err := time.ParseDuration(cfg.Global.HistoryInterval)
			if err != nil {
				return fmt.Errorf("global.history_interval: invalid duration %q: %w", cfg.Global.HistoryInterval, err)
			}
			if d < 2*time.Second {
				return fmt.Errorf("global.history_interval: minimum is 2s, got %q", cfg.Global.HistoryInterval)
			}
		}
		if cfg.Global.HistoryProcs < 0 {
			return fmt.Errorf("global.history_procs: must be >= 0, got %d", cfg.Global.HistoryProcs)
		}
	}

	if cfg.Layout != nil {
		for i, row := range cfg.Layout.Rows {
			if row.Weight < 0 {
				return fmt.Errorf("layout.row[%d]: weight must be non-negative, got %d", i, row.Weight)
			}
			if len(row.Cols) == 0 {
				return fmt.Errorf("layout.row[%d]: must have at least one col", i)
			}
			for j, col := range row.Cols {
				if col.Weight < 0 {
					return fmt.Errorf("layout.row[%d].col[%d]: weight must be non-negative, got %d", i, j, col.Weight)
				}
				if col.Widget == "" {
					return fmt.Errorf("layout.row[%d].col[%d]: widget is required", i, j)
				}
				if !KnownWidgets[col.Widget] {
					return fmt.Errorf("layout.row[%d].col[%d]: unknown widget %q (known: summary, procs, nettop)", i, j, col.Widget)
				}
			}
		}
	}

	return nil
}
