package ui

import (
	"sort"

	"github.com/brajesh/termdash/internal/metrics"
)

// ProcessGroup represents a group of processes with the same name.
type ProcessGroup struct {
	Name       string
	User       string // from first process in group
	Processes  []metrics.ProcessInfo
	CPUPercent float64  // sum
	MemPercent float32  // sum
	MemRSS     uint64   // sum
	ConnCount  int      // sum
}

// DisplayRow represents a single row in the process table,
// which can be either a group header or an individual process.
type DisplayRow struct {
	IsGroup  bool
	Group    *ProcessGroup        // non-nil if IsGroup or is child of group
	Process  *metrics.ProcessInfo // non-nil if single process or child
	Depth    int                  // 0 = top level, 1 = child of expanded group
	Expanded bool                 // only meaningful if IsGroup
}

// GroupProcesses groups processes by name and computes aggregates.
func GroupProcesses(procs []metrics.ProcessInfo, connCounts map[int32]int) []ProcessGroup {
	// Group by name
	groupMap := make(map[string]*ProcessGroup)
	var order []string

	for _, p := range procs {
		// Apply connection count
		proc := p
		if c, ok := connCounts[proc.PID]; ok {
			proc.ConnCount = c
		}

		if g, ok := groupMap[proc.Name]; ok {
			g.Processes = append(g.Processes, proc)
			g.CPUPercent += proc.CPUPercent
			g.MemPercent += proc.MemPercent
			g.MemRSS += proc.MemRSS
			if proc.ConnCount >= 0 {
				g.ConnCount += proc.ConnCount
			}
		} else {
			order = append(order, proc.Name)
			groupMap[proc.Name] = &ProcessGroup{
				Name:       proc.Name,
				User:       proc.User,
				Processes:  []metrics.ProcessInfo{proc},
				CPUPercent: proc.CPUPercent,
				MemPercent: proc.MemPercent,
				MemRSS:     proc.MemRSS,
				ConnCount:  proc.ConnCount,
			}
		}
	}

	// Build result in order of first appearance
	groups := make([]ProcessGroup, 0, len(order))
	for _, name := range order {
		groups = append(groups, *groupMap[name])
	}

	return groups
}

// SortGroups sorts groups by the specified column and direction.
func SortGroups(groups []ProcessGroup, sortCol SortColumn, sortAsc bool) {
	less := func(i, j int) bool {
		switch sortCol {
		case SortByCPU:
			return groups[i].CPUPercent > groups[j].CPUPercent
		case SortByMem:
			return groups[i].MemPercent > groups[j].MemPercent
		case SortByPID:
			// Sort by process count for groups
			return len(groups[i].Processes) > len(groups[j].Processes)
		case SortByName:
			return groups[i].Name < groups[j].Name
		case SortByConns:
			return groups[i].ConnCount > groups[j].ConnCount
		default:
			return groups[i].CPUPercent > groups[j].CPUPercent
		}
	}

	if sortAsc {
		sort.Slice(groups, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.Slice(groups, less)
	}
}

// BuildDisplayRows builds the flat list of rows for rendering,
// respecting which groups are expanded.
func BuildDisplayRows(
	groups []ProcessGroup,
	expandedGroups map[string]bool,
	sortCol SortColumn,
	sortAsc bool,
) []DisplayRow {
	// Sort groups first
	SortGroups(groups, sortCol, sortAsc)

	var rows []DisplayRow

	for i := range groups {
		g := &groups[i]
		expanded := expandedGroups[g.Name]

		if len(g.Processes) == 1 {
			// Single process - show as regular row, not as group
			proc := g.Processes[0]
			rows = append(rows, DisplayRow{
				IsGroup: false,
				Group:   g,
				Process: &proc,
				Depth:   0,
			})
		} else {
			// Multi-process group
			rows = append(rows, DisplayRow{
				IsGroup:  true,
				Group:    g,
				Process:  nil,
				Depth:    0,
				Expanded: expanded,
			})

			if expanded {
				// Sort children by CPU descending
				children := make([]metrics.ProcessInfo, len(g.Processes))
				copy(children, g.Processes)
				sort.Slice(children, func(i, j int) bool {
					return children[i].CPUPercent > children[j].CPUPercent
				})

				for j := range children {
					proc := children[j]
					rows = append(rows, DisplayRow{
						IsGroup: false,
						Group:   g,
						Process: &proc,
						Depth:   1,
					})
				}
			}
		}
	}

	return rows
}
