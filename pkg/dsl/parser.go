package dsl

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// LoadConfig parses a .td config file and returns the validated Config.
// If the file does not exist, it returns DefaultConfig with no error.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg, err := Parse(data, path)
	if err != nil {
		return nil, err
	}

	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	applyDefaults(cfg)
	return cfg, nil
}

// Parse parses raw HCL bytes into a Config struct.
func Parse(src []byte, filename string) (*Config, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse error: %s", diags.Error())
	}

	var cfg Config
	diags = gohcl.DecodeBody(file.Body, nil, &cfg)
	if diags.HasErrors() {
		return nil, fmt.Errorf("decode error: %s", diags.Error())
	}

	return &cfg, nil
}

// ParseDuration parses a Go duration string (e.g. "2s", "500ms", "5m").
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// applyDefaults fills in zero-value fields with sensible defaults.
func applyDefaults(cfg *Config) {
	if cfg.Global == nil {
		cfg.Global = &GlobalBlock{}
	}
	if cfg.Global.Refresh == "" {
		cfg.Global.Refresh = "2s"
	}
	if cfg.Global.History == "" {
		cfg.Global.History = "24h"
	}
	if cfg.Global.HistoryInterval == "" {
		cfg.Global.HistoryInterval = "10s"
	}
	if cfg.Global.HistoryProcs == 0 {
		cfg.Global.HistoryProcs = 50
	}
	if cfg.Layout == nil {
		def := DefaultConfig()
		cfg.Layout = def.Layout
	}
	for i := range cfg.Layout.Rows {
		if cfg.Layout.Rows[i].Weight == 0 {
			cfg.Layout.Rows[i].Weight = 1
		}
		for j := range cfg.Layout.Rows[i].Cols {
			if cfg.Layout.Rows[i].Cols[j].Weight == 0 {
				cfg.Layout.Rows[i].Cols[j].Weight = 1
			}
		}
	}
}
