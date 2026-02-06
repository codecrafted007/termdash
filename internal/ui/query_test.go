package ui

import (
	"testing"

	"github.com/brajesh/termdash/internal/metrics"
)

func TestParseQuery_Empty(t *testing.T) {
	node, err := ParseQuery("")
	if err != nil {
		t.Errorf("expected no error for empty query, got %v", err)
	}
	if node != nil {
		t.Errorf("expected nil node for empty query")
	}
}

func TestParseQuery_SimpleExpression(t *testing.T) {
	tests := []struct {
		input    string
		field    string
		op       FilterOp
		value    string
		wantErr  bool
	}{
		{"name = chrome", "name", OpEq, "chrome", false},
		{"cpu > 50", "cpu", OpGt, "50", false},
		{"mem < 10", "mem", OpLt, "10", false},
		{"user != root", "user", OpNeq, "root", false},
		{"pid >= 1000", "pid", OpGte, "1000", false},
		{"conn <= 5", "conn", OpLte, "5", false},
		{"name ~ java", "name", OpContains, "java", false},
		{"invalid = foo", "", 0, "", true}, // unknown field
		{"name", "", 0, "", true},          // missing operator
		{"name =", "", 0, "", true},        // missing value
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := ParseQuery(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tt.input, err)
				return
			}
			if node == nil || node.Expr == nil {
				t.Errorf("expected non-nil node with expression for %q", tt.input)
				return
			}
			if node.Expr.Field != tt.field {
				t.Errorf("expected field %q, got %q", tt.field, node.Expr.Field)
			}
			if node.Expr.Op != tt.op {
				t.Errorf("expected op %v, got %v", tt.op, node.Expr.Op)
			}
			if node.Expr.Value != tt.value {
				t.Errorf("expected value %q, got %q", tt.value, node.Expr.Value)
			}
		})
	}
}

func TestParseQuery_AndExpression(t *testing.T) {
	node, err := ParseQuery("cpu > 10 and mem < 50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.Op != "and" {
		t.Errorf("expected 'and' operator, got %q", node.Op)
	}
	if node.Left == nil || node.Left.Expr == nil {
		t.Fatal("expected left expression")
	}
	if node.Left.Expr.Field != "cpu" {
		t.Errorf("expected left field 'cpu', got %q", node.Left.Expr.Field)
	}
	if node.Right == nil || node.Right.Expr == nil {
		t.Fatal("expected right expression")
	}
	if node.Right.Expr.Field != "mem" {
		t.Errorf("expected right field 'mem', got %q", node.Right.Expr.Field)
	}
}

func TestParseQuery_OrExpression(t *testing.T) {
	node, err := ParseQuery("name ~ java or name ~ python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node == nil {
		t.Fatal("expected non-nil node")
	}
	if node.Op != "or" {
		t.Errorf("expected 'or' operator, got %q", node.Op)
	}
}

func TestParseQuery_QuotedValue(t *testing.T) {
	node, err := ParseQuery(`name ~ "my process"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node == nil || node.Expr == nil {
		t.Fatal("expected non-nil node with expression")
	}
	if node.Expr.Value != "my process" {
		t.Errorf("expected value 'my process', got %q", node.Expr.Value)
	}
}

func TestFilterNode_MatchProcess(t *testing.T) {
	proc := metrics.ProcessInfo{
		PID:        1234,
		Name:       "chrome",
		User:       "john",
		CPUPercent: 25.5,
		MemPercent: 10.2,
	}
	connCount := 5

	tests := []struct {
		query   string
		want    bool
	}{
		{"name = chrome", true},
		{"name = firefox", false},
		{"name ~ chro", true},
		{"name ~ fire", false},
		{"user = john", true},
		{"user = root", false},
		{"pid = 1234", true},
		{"pid = 5678", false},
		{"cpu > 20", true},
		{"cpu > 30", false},
		{"cpu < 30", true},
		{"cpu >= 25.5", true},
		{"cpu <= 25.5", true},
		{"mem > 5", true},
		{"mem < 5", false},
		{"conn >= 5", true},
		{"conn > 5", false},
		{"name = chrome and cpu > 20", true},
		{"name = chrome and cpu > 30", false},
		{"name = firefox or cpu > 20", true},
		{"name = firefox or cpu > 30", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			node, err := ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			got := node.MatchProcess(proc, connCount)
			if got != tt.want {
				t.Errorf("MatchProcess(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestSimpleSearch(t *testing.T) {
	proc := metrics.ProcessInfo{
		PID:  1234,
		Name: "MyProcess",
	}

	tests := []struct {
		query string
		want  bool
	}{
		{"", true},
		{"MyProcess", true},
		{"myprocess", true}, // case insensitive
		{"MYPROCESS", true},
		{"Process", true},   // substring
		{"notfound", false},
		{"1234", true},      // matches PID
		{"234", true},       // PID substring
		{"5678", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := SimpleSearch(proc, tt.query)
			if got != tt.want {
				t.Errorf("SimpleSearch(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestFilterOp_String(t *testing.T) {
	tests := []struct {
		op   FilterOp
		want string
	}{
		{OpEq, "="},
		{OpNeq, "!="},
		{OpGt, ">"},
		{OpLt, "<"},
		{OpGte, ">="},
		{OpLte, "<="},
		{OpContains, "~"},
	}

	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("FilterOp(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestParseQuery_CaseInsensitiveFieldsAndOperators(t *testing.T) {
	// Fields should be case-insensitive
	node, err := ParseQuery("NAME = test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Expr.Field != "name" {
		t.Errorf("expected field 'name', got %q", node.Expr.Field)
	}

	// AND/OR should be case-insensitive
	node, err = ParseQuery("cpu > 10 AND mem < 50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Op != "and" {
		t.Errorf("expected 'and' operator, got %q", node.Op)
	}

	node, err = ParseQuery("name ~ java OR name ~ python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Op != "or" {
		t.Errorf("expected 'or' operator, got %q", node.Op)
	}
}

func TestNilFilterNode_MatchProcess(t *testing.T) {
	var node *FilterNode
	proc := metrics.ProcessInfo{PID: 1, Name: "test"}

	// nil filter should match everything
	if !node.MatchProcess(proc, 0) {
		t.Error("nil FilterNode should match all processes")
	}
}
