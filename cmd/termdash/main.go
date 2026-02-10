package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/brajesh/termdash/internal/history"
	"github.com/brajesh/termdash/internal/ui"
	"github.com/brajesh/termdash/pkg/dsl"
	tea "github.com/charmbracelet/bubbletea"
)

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "termdash", "config.td")
}

func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "termdash", "history.db")
}

func main() {
	configPath := flag.String("config", defaultConfigPath(), "path to .td config file")
	flag.StringVar(configPath, "c", defaultConfigPath(), "path to .td config file (shorthand)")
	flag.Parse()

	cfg, err := dsl.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	model := ui.NewWithConfig(cfg)

	// Set up history store if not disabled.
	historyEnabled := cfg.Global != nil && cfg.Global.History != "0"
	var store *history.Store
	if historyEnabled {
		dbPath := defaultDBPath()
		if cfg.Global != nil && cfg.Global.DBPath != "" {
			dbPath = cfg.Global.DBPath
		}

		var retention, interval time.Duration
		var maxProcs int

		if cfg.Global != nil {
			retention, _ = time.ParseDuration(cfg.Global.History)
			interval, _ = time.ParseDuration(cfg.Global.HistoryInterval)
			maxProcs = cfg.Global.HistoryProcs
		}
		if retention == 0 {
			retention = 24 * time.Hour
		}
		if interval == 0 {
			interval = 10 * time.Second
		}
		if maxProcs == 0 {
			maxProcs = 50
		}

		store, err = history.Open(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: history unavailable: %v\n", err)
		} else {
			defer store.Close()
			model.SetHistoryStore(store, retention, interval, maxProcs)
		}
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
