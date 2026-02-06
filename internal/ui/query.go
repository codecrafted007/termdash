package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/brajesh/termdash/internal/metrics"
)

// QueryType distinguishes simple search from field queries.
type QueryType int

const (
	QuerySearch QueryType = iota // "/" simple search
	QueryFilter                  // "Q" field-based DSL
)

// FilterOp represents comparison operators.
type FilterOp int

const (
	OpEq       FilterOp = iota // =
	OpNeq                      // !=
	OpGt                       // >
	OpLt                       // <
	OpGte                      // >=
	OpLte                      // <=
	OpContains                 // ~
)

func (op FilterOp) String() string {
	switch op {
	case OpEq:
		return "="
	case OpNeq:
		return "!="
	case OpGt:
		return ">"
	case OpLt:
		return "<"
	case OpGte:
		return ">="
	case OpLte:
		return "<="
	case OpContains:
		return "~"
	default:
		return "?"
	}
}

// FilterExpr represents a single filter condition.
type FilterExpr struct {
	Field string
	Op    FilterOp
	Value string
}

// FilterNode represents a parsed query (supports AND/OR).
type FilterNode struct {
	Expr  *FilterExpr // leaf node (nil for branch nodes)
	Op    string      // "and" or "or" for branch nodes
	Left  *FilterNode
	Right *FilterNode
}

// validFields are the fields that can be queried.
var validFields = map[string]bool{
	"name": true,
	"user": true,
	"pid":  true,
	"cpu":  true,
	"mem":  true,
	"conn": true,
}

// ParseQuery parses a query string into a FilterNode tree.
// Returns nil, nil for empty input.
func ParseQuery(input string) (*FilterNode, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}

	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, nil
	}

	p := &parser{tokens: tokens, pos: 0}
	node, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if p.pos < len(p.tokens) {
		return nil, fmt.Errorf("unexpected token: %s", p.tokens[p.pos])
	}

	return node, nil
}

// tokenize splits input into tokens, handling quoted strings.
func tokenize(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	runes := []rune(input)
	i := 0

	for i < len(runes) {
		r := runes[i]

		// Handle quoted strings
		if r == '"' || r == '\'' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			quoteChar := r
			i++
			for i < len(runes) && runes[i] != quoteChar {
				current.WriteRune(runes[i])
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("unterminated quote")
			}
			tokens = append(tokens, current.String())
			current.Reset()
			i++ // skip closing quote
			continue
		}

		// Handle whitespace
		if unicode.IsSpace(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			i++
			continue
		}

		// Handle multi-char operators: !=, >=, <=
		if r == '!' || r == '>' || r == '<' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			if i+1 < len(runes) && runes[i+1] == '=' {
				tokens = append(tokens, string(r)+"=")
				i += 2 // skip both characters
			} else {
				tokens = append(tokens, string(r))
				i++
			}
			continue
		}

		// Handle single-char operators: =, ~
		if r == '=' || r == '~' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(r))
			i++
			continue
		}

		// Regular character
		current.WriteRune(r)
		i++
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

type parser struct {
	tokens []string
	pos    int
}

func (p *parser) peek() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.pos]
}

func (p *parser) consume() string {
	if p.pos >= len(p.tokens) {
		return ""
	}
	t := p.tokens[p.pos]
	p.pos++
	return t
}

// parseOr handles: expr (or expr)*
func (p *parser) parseOr() (*FilterNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for strings.EqualFold(p.peek(), "or") {
		p.consume() // consume "or"
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &FilterNode{
			Op:    "or",
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parseAnd handles: primary (and primary)*
func (p *parser) parseAnd() (*FilterNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for strings.EqualFold(p.peek(), "and") {
		p.consume() // consume "and"
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = &FilterNode{
			Op:    "and",
			Left:  left,
			Right: right,
		}
	}

	return left, nil
}

// parsePrimary handles: field op value
func (p *parser) parsePrimary() (*FilterNode, error) {
	field := p.consume()
	if field == "" {
		return nil, fmt.Errorf("expected field name")
	}

	fieldLower := strings.ToLower(field)
	if !validFields[fieldLower] {
		return nil, fmt.Errorf("unknown field: %s (valid: name, user, pid, cpu, mem, conn)", field)
	}

	opStr := p.consume()
	if opStr == "" {
		return nil, fmt.Errorf("expected operator after %s", field)
	}

	op, err := parseOp(opStr)
	if err != nil {
		return nil, err
	}

	value := p.consume()
	if value == "" {
		return nil, fmt.Errorf("expected value after %s %s", field, opStr)
	}

	return &FilterNode{
		Expr: &FilterExpr{
			Field: fieldLower,
			Op:    op,
			Value: value,
		},
	}, nil
}

func parseOp(s string) (FilterOp, error) {
	switch s {
	case "=":
		return OpEq, nil
	case "!=":
		return OpNeq, nil
	case ">":
		return OpGt, nil
	case "<":
		return OpLt, nil
	case ">=":
		return OpGte, nil
	case "<=":
		return OpLte, nil
	case "~":
		return OpContains, nil
	default:
		return 0, fmt.Errorf("unknown operator: %s (valid: =, !=, >, <, >=, <=, ~)", s)
	}
}

// MatchProcess returns true if the process matches the filter.
func (f *FilterNode) MatchProcess(p metrics.ProcessInfo, connCount int) bool {
	if f == nil {
		return true
	}

	// Branch node
	if f.Expr == nil {
		leftMatch := f.Left.MatchProcess(p, connCount)
		rightMatch := f.Right.MatchProcess(p, connCount)

		switch f.Op {
		case "and":
			return leftMatch && rightMatch
		case "or":
			return leftMatch || rightMatch
		default:
			return false
		}
	}

	// Leaf node - evaluate expression
	return f.Expr.evaluate(p, connCount)
}

func (e *FilterExpr) evaluate(p metrics.ProcessInfo, connCount int) bool {
	switch e.Field {
	case "name":
		return e.compareString(p.Name)
	case "user":
		return e.compareString(p.User)
	case "pid":
		return e.compareInt(int64(p.PID))
	case "cpu":
		return e.compareFloat(p.CPUPercent)
	case "mem":
		return e.compareFloat(float64(p.MemPercent))
	case "conn":
		c := connCount
		if c < 0 {
			c = 0
		}
		return e.compareInt(int64(c))
	default:
		return false
	}
}

func (e *FilterExpr) compareString(actual string) bool {
	actualLower := strings.ToLower(actual)
	valueLower := strings.ToLower(e.Value)

	switch e.Op {
	case OpEq:
		return actualLower == valueLower
	case OpNeq:
		return actualLower != valueLower
	case OpContains:
		return strings.Contains(actualLower, valueLower)
	case OpGt:
		return actualLower > valueLower
	case OpLt:
		return actualLower < valueLower
	case OpGte:
		return actualLower >= valueLower
	case OpLte:
		return actualLower <= valueLower
	default:
		return false
	}
}

func (e *FilterExpr) compareInt(actual int64) bool {
	expected, err := strconv.ParseInt(e.Value, 10, 64)
	if err != nil {
		// If value isn't a valid int, try string comparison
		return e.compareString(strconv.FormatInt(actual, 10))
	}

	switch e.Op {
	case OpEq:
		return actual == expected
	case OpNeq:
		return actual != expected
	case OpGt:
		return actual > expected
	case OpLt:
		return actual < expected
	case OpGte:
		return actual >= expected
	case OpLte:
		return actual <= expected
	case OpContains:
		return strings.Contains(strconv.FormatInt(actual, 10), e.Value)
	default:
		return false
	}
}

func (e *FilterExpr) compareFloat(actual float64) bool {
	expected, err := strconv.ParseFloat(e.Value, 64)
	if err != nil {
		return false
	}

	switch e.Op {
	case OpEq:
		return actual == expected
	case OpNeq:
		return actual != expected
	case OpGt:
		return actual > expected
	case OpLt:
		return actual < expected
	case OpGte:
		return actual >= expected
	case OpLte:
		return actual <= expected
	case OpContains:
		// Contains doesn't make sense for floats, treat as >=
		return actual >= expected
	default:
		return false
	}
}

// SimpleSearch returns true if the process matches a simple text search.
// Matches against name (case-insensitive) or PID.
func SimpleSearch(p metrics.ProcessInfo, query string) bool {
	if query == "" {
		return true
	}

	queryLower := strings.ToLower(query)

	// Check if name contains query
	if strings.Contains(strings.ToLower(p.Name), queryLower) {
		return true
	}

	// Check if query is a number and matches PID
	if pid, err := strconv.ParseInt(query, 10, 32); err == nil {
		if int32(pid) == p.PID {
			return true
		}
	}

	// Also check if PID contains the query string
	pidStr := strconv.FormatInt(int64(p.PID), 10)
	if strings.Contains(pidStr, query) {
		return true
	}

	return false
}
