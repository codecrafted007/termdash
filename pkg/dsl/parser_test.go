package dsl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMinimalConfig(t *testing.T) {
	src := []byte(`
global {
  refresh = "2s"
  title   = "my workstation"
}

layout {
  row {
    weight = 1
    col {
      weight = 3
      widget = "summary"
    }
    col {
      weight = 2
      widget = "nettop"
    }
  }
  row {
    weight = 3
    col {
      weight = 1
      widget = "procs"
    }
  }
}
`)
	cfg, err := Parse(src, "test.td")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Global.Refresh != "2s" {
		t.Errorf("refresh = %q, want %q", cfg.Global.Refresh, "2s")
	}
	if cfg.Global.Title != "my workstation" {
		t.Errorf("title = %q, want %q", cfg.Global.Title, "my workstation")
	}
	if len(cfg.Layout.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(cfg.Layout.Rows))
	}
	if cfg.Layout.Rows[0].Weight != 1 {
		t.Errorf("row[0].weight = %d, want 1", cfg.Layout.Rows[0].Weight)
	}
	if len(cfg.Layout.Rows[0].Cols) != 2 {
		t.Fatalf("row[0].cols = %d, want 2", len(cfg.Layout.Rows[0].Cols))
	}
	if cfg.Layout.Rows[0].Cols[0].Widget != "summary" {
		t.Errorf("row[0].col[0].widget = %q, want %q", cfg.Layout.Rows[0].Cols[0].Widget, "summary")
	}
	if cfg.Layout.Rows[1].Cols[0].Widget != "procs" {
		t.Errorf("row[1].col[0].widget = %q, want %q", cfg.Layout.Rows[1].Cols[0].Widget, "procs")
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("Validate failed: %v", err)
	}
}

func TestParseMissingOptionalFields(t *testing.T) {
	src := []byte(`
layout {
  row {
    col {
      widget = "procs"
    }
  }
}
`)
	cfg, err := Parse(src, "test.td")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	applyDefaults(cfg)

	if cfg.Global.Refresh != "2s" {
		t.Errorf("default refresh = %q, want %q", cfg.Global.Refresh, "2s")
	}
	if cfg.Layout.Rows[0].Weight != 1 {
		t.Errorf("default row weight = %d, want 1", cfg.Layout.Rows[0].Weight)
	}
	if cfg.Layout.Rows[0].Cols[0].Weight != 1 {
		t.Errorf("default col weight = %d, want 1", cfg.Layout.Rows[0].Cols[0].Weight)
	}
}

func TestValidateInvalidWidget(t *testing.T) {
	cfg := &Config{
		Layout: &LayoutBlock{
			Rows: []RowBlock{
				{Weight: 1, Cols: []ColBlock{{Weight: 1, Widget: "bogus"}}},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for unknown widget")
	}
}

func TestValidateInvalidDuration(t *testing.T) {
	cfg := &Config{
		Global: &GlobalBlock{Refresh: "not-a-duration"},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for bad duration")
	}
}

func TestValidateEmptyRow(t *testing.T) {
	cfg := &Config{
		Layout: &LayoutBlock{
			Rows: []RowBlock{
				{Weight: 1, Cols: nil},
			},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for empty row")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path/config.td")
	if err != nil {
		t.Fatalf("LoadConfig should not error for missing file: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}
	if cfg.Global.Refresh != "2s" {
		t.Errorf("default refresh = %q, want %q", cfg.Global.Refresh, "2s")
	}
	if len(cfg.Layout.Rows) == 0 {
		t.Error("expected non-empty default layout")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.td")
	err := os.WriteFile(path, []byte(`
global {
  refresh = "5s"
}
layout {
  row {
    col { widget = "procs" }
  }
}
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Global.Refresh != "5s" {
		t.Errorf("refresh = %q, want %q", cfg.Global.Refresh, "5s")
	}
}

func TestDefaultConfigMatchesCurrentBehavior(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Global.Refresh != "2s" {
		t.Errorf("default refresh = %q, want %q", cfg.Global.Refresh, "2s")
	}
	if len(cfg.Layout.Rows) != 2 {
		t.Fatalf("default layout rows = %d, want 2", len(cfg.Layout.Rows))
	}
	// First row: summary + nettop
	if len(cfg.Layout.Rows[0].Cols) != 2 {
		t.Fatalf("default row[0] cols = %d, want 2", len(cfg.Layout.Rows[0].Cols))
	}
	if cfg.Layout.Rows[0].Cols[0].Widget != "summary" {
		t.Errorf("row[0].col[0].widget = %q, want %q", cfg.Layout.Rows[0].Cols[0].Widget, "summary")
	}
	if cfg.Layout.Rows[0].Cols[1].Widget != "nettop" {
		t.Errorf("row[0].col[1].widget = %q, want %q", cfg.Layout.Rows[0].Cols[1].Widget, "nettop")
	}
	// Second row: procs
	if len(cfg.Layout.Rows[1].Cols) != 1 {
		t.Fatalf("default row[1] cols = %d, want 1", len(cfg.Layout.Rows[1].Cols))
	}
	if cfg.Layout.Rows[1].Cols[0].Widget != "procs" {
		t.Errorf("row[1].col[0].widget = %q, want %q", cfg.Layout.Rows[1].Cols[0].Widget, "procs")
	}

	if err := Validate(cfg); err != nil {
		t.Errorf("default config fails validation: %v", err)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2s", "2s"},
		{"500ms", "500ms"},
		{"5m", "5m0s"},
		{"1h30m", "1h30m0s"},
	}
	for _, tt := range tests {
		d, err := ParseDuration(tt.input)
		if err != nil {
			t.Errorf("ParseDuration(%q) error: %v", tt.input, err)
			continue
		}
		if d.String() != tt.want {
			t.Errorf("ParseDuration(%q) = %s, want %s", tt.input, d.String(), tt.want)
		}
	}
}

func TestParseWithStubBlocks(t *testing.T) {
	src := []byte(`
global {
  refresh = "2s"
}
layout {
  row {
    col { widget = "procs" }
  }
}
metric "cpu_avg" {
  expr = "avg(cpu.cores)"
}
alert "high_cpu" {
  when = "cpu > 90"
  action = "flash"
}
source "docker" {
  type = "exec"
  cmd  = "docker stats --no-stream"
}
`)
	cfg, err := Parse(src, "test.td")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(cfg.Metrics) != 1 {
		t.Errorf("metrics = %d, want 1", len(cfg.Metrics))
	}
	if len(cfg.Alerts) != 1 {
		t.Errorf("alerts = %d, want 1", len(cfg.Alerts))
	}
	if len(cfg.Sources) != 1 {
		t.Errorf("sources = %d, want 1", len(cfg.Sources))
	}
	if cfg.Metrics[0].Name != "cpu_avg" {
		t.Errorf("metric name = %q, want %q", cfg.Metrics[0].Name, "cpu_avg")
	}
}
