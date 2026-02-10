package dsl

import "github.com/hashicorp/hcl/v2"

// Config is the top-level structure for a .td config file.
type Config struct {
	Global  *GlobalBlock  `hcl:"global,block"`
	Layout  *LayoutBlock  `hcl:"layout,block"`
	Metrics []MetricBlock `hcl:"metric,block"`
	Alerts  []AlertBlock  `hcl:"alert,block"`
	Sources []SourceBlock `hcl:"source,block"`
}

// GlobalBlock holds dashboard-wide settings.
type GlobalBlock struct {
	Refresh         string `hcl:"refresh,optional"`
	Title           string `hcl:"title,optional"`
	History         string `hcl:"history,optional"`          // retention period, e.g. "24h", "7d", "0" to disable
	HistoryInterval string `hcl:"history_interval,optional"` // write frequency, e.g. "10s"
	HistoryProcs    int    `hcl:"history_procs,optional"`    // max processes per snapshot, 0 = all
	DBPath          string `hcl:"db_path,optional"`          // custom path for history DB
}

// LayoutBlock describes the row/col grid layout.
type LayoutBlock struct {
	Rows []RowBlock `hcl:"row,block"`
}

// RowBlock is a horizontal row in the layout grid.
type RowBlock struct {
	Weight int        `hcl:"weight,optional"`
	Cols   []ColBlock `hcl:"col,block"`
}

// ColBlock is a column within a row, holding a single widget.
type ColBlock struct {
	Weight int    `hcl:"weight,optional"`
	Widget string `hcl:"widget"`
}

// Stub blocks for milestone 2 — parsed but not evaluated.

type MetricBlock struct {
	Name   string   `hcl:"name,label"`
	Remain hcl.Body `hcl:",remain"`
}

type AlertBlock struct {
	Name   string   `hcl:"name,label"`
	Remain hcl.Body `hcl:",remain"`
}

type SourceBlock struct {
	Name   string   `hcl:"name,label"`
	Remain hcl.Body `hcl:",remain"`
}
