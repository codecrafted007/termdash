# Process Grouping by Application — Implementation Plan

## Goal
Add `p` key to toggle between flat process view and grouped-by-name view.
In grouped view, processes with the same name are aggregated into a single row
showing combined CPU%, MEM%, RSS, and CONN. Groups can be expanded/collapsed
with Enter to show individual PIDs.

## Behavior

### Flat View (current, default)
```
  PID     NAME           USER        CPU%   MEM%      RSS  CONN
  1234    chrome         brajesh      2.1    0.4     120M    12
  1235    chrome         brajesh      1.8    0.3      98M     8
  1236    chrome         brajesh      0.9    0.2      64M     4
  5678    code           brajesh      3.2    1.2     256M    15
```

### Grouped View (press `p`)
```
  PID     NAME           USER        CPU%   MEM%      RSS  CONN
▶ (38)   chrome         brajesh     12.4    8.2     3.1G   147
▶ (15)   code           brajesh      8.1    5.4     2.1G    23
  1       Terminal       brajesh      0.3    0.1      45M     2
```

- `▶` prefix indicates collapsed group with count in PID column
- Single-process groups show normally (no indicator)
- Sorting applies to aggregate values

### Expanded Group (press Enter on group row)
```
  PID     NAME           USER        CPU%   MEM%      RSS  CONN
▼ (38)   chrome         brajesh     12.4    8.2     3.1G   147
    1234  ├─chrome       brajesh      2.1    0.4     120M    12
    1235  ├─chrome       brajesh      1.8    0.3      98M     8
    1236  └─chrome       brajesh      0.9    0.2      64M     4
▶ (15)   code           brajesh      8.1    5.4     2.1G    23
```

- `▼` indicates expanded group
- Children indented with tree characters
- Enter on child row → detail view for that PID
- Enter on expanded group → collapse it

---

## Data Structures

### ProcessGroup (new type in model.go or separate file)
```go
type ProcessGroup struct {
    Name       string
    User       string             // from first process in group
    Processes  []metrics.ProcessInfo
    CPUPercent float64            // sum
    MemPercent float32            // sum
    MemRSS     uint64             // sum
    ConnCount  int                // sum
}
```

### DisplayRow (for unified cursor navigation)
```go
type DisplayRow struct {
    IsGroup    bool               // true = group header, false = process
    Group      *ProcessGroup      // non-nil if IsGroup or is child of group
    Process    *metrics.ProcessInfo // non-nil if single process or child
    Depth      int                // 0 = top level, 1 = child of expanded group
}
```

---

## Model State Changes

Add to Model struct:
```go
// Process grouping
groupedView    bool              // toggle with 'p' key
expandedGroups map[string]bool   // group name → expanded
```

Initialize in New():
```go
expandedGroups: make(map[string]bool),
```

---

## Implementation Steps

### Step 1: Add ProcessGroup type and grouping logic

**File: `internal/ui/grouping.go`** (new)

```go
package ui

import "github.com/brajesh/termdash/internal/metrics"

type ProcessGroup struct {
    Name       string
    User       string
    Processes  []metrics.ProcessInfo
    CPUPercent float64
    MemPercent float32
    MemRSS     uint64
    ConnCount  int
}

type DisplayRow struct {
    IsGroup  bool
    Group    *ProcessGroup
    Process  *metrics.ProcessInfo
    Depth    int
}

// GroupProcesses groups processes by name and computes aggregates.
func GroupProcesses(procs []metrics.ProcessInfo, connCounts map[int32]int) []ProcessGroup {
    // ... implementation
}

// BuildDisplayRows builds the flat list of rows for rendering,
// respecting which groups are expanded.
func BuildDisplayRows(
    groups []ProcessGroup,
    expandedGroups map[string]bool,
    sortCol SortColumn,
    sortAsc bool,
) []DisplayRow {
    // ... implementation
}
```

### Step 2: Add model state and `p` key handler

**File: `internal/ui/model.go`**

- Add `groupedView bool` and `expandedGroups map[string]bool` fields
- Initialize `expandedGroups` in `New()`
- Add `"p"` case in `handleDashboardKey` to toggle `groupedView`
- Modify cursor clamping in `snapshotMsg` handler to use display row count

### Step 3: Modify sortedProcesses → displayRows

**File: `internal/ui/model.go`**

- When `groupedView == false`: current behavior (return []ProcessInfo)
- When `groupedView == true`:
  - Call `GroupProcesses()` to build groups
  - Call `BuildDisplayRows()` with expanded state
  - Return []DisplayRow

Actually, we need a unified approach. Change to always use DisplayRow
internally when in grouped mode, but keep current path for flat mode
to minimize changes.

New method:
```go
func (m Model) displayRows() []DisplayRow
```

### Step 4: Modify Enter key behavior

**File: `internal/ui/model.go`** — `handleDashboardKey`

```go
case "enter":
    if m.groupedView {
        rows := m.displayRows()
        if m.cursor < len(rows) {
            row := rows[m.cursor]
            if row.IsGroup {
                // Toggle expand/collapse
                name := row.Group.Name
                m.expandedGroups[name] = !m.expandedGroups[name]
                return m, nil
            } else {
                // Open detail for this process
                m.viewState = ViewProcessDetail
                m.detailPID = row.Process.PID
                // ... rest of detail setup
            }
        }
    } else {
        // current behavior
    }
```

### Step 5: Update RenderProcessTable to handle grouped mode

**File: `internal/ui/panels/proctable.go`**

Option A: Pass DisplayRow slice instead of ProcessInfo slice
Option B: New function RenderGroupedProcessTable

Going with Option A — modify signature:
```go
func RenderProcessTable(
    rows []DisplayRow,  // unified row type
    cursor, scrollOffset, visibleRows int,
    sortCol SortColumn, sortAsc bool,
    width int,
) string
```

This requires defining DisplayRow in a shared location (panels package or
moving to metrics package).

**Alternative:** Keep proctable.go simple, create a new struct in panels:

```go
type TableRow struct {
    PID        string  // "(38)" for group, "1234" for process
    Name       string
    User       string
    CPUPercent float64
    MemPercent float32
    MemRSS     uint64
    ConnCount  int
    IsGroup    bool
    Expanded   bool
    Depth      int
}
```

Model converts DisplayRow → TableRow before passing to renderer.

### Step 6: Update help panel

**File: `internal/ui/panels/help.go`**

- Compact: add `p:group`
- Expanded: add `{"p", "Toggle process grouping"}`

### Step 7: Update cursor clamping

When switching views or when snapshot updates, cursor must be clamped
to the new row count (which differs between flat and grouped modes).

---

## Files Created (1)
- `internal/ui/grouping.go` — ProcessGroup, DisplayRow, GroupProcesses, BuildDisplayRows

## Files Modified (3)
- `internal/ui/model.go` — state fields, p key, enter key, displayRows method
- `internal/ui/panels/proctable.go` — TableRow struct, updated rendering
- `internal/ui/panels/help.go` — p key documentation

## Edge Cases
- Empty process list → no groups
- All processes have unique names → each group has 1 member (no indicator)
- Group becomes empty between snapshots → remove from expandedGroups
- Cursor on expanded child when group is collapsed → clamp cursor
- Sort column changes → groups re-sorted, cursor may need clamping

## Verification
- `go build ./...` compiles
- `go test ./...` passes
- Press `p` → view toggles, groups visible with counts
- Press `p` again → back to flat view
- In grouped view, Enter on group → expands/collapses
- In grouped view, Enter on child → opens detail
- Sorting works on aggregates in grouped mode
- j/k navigation works through groups and children
- Resize terminal → no crash
- CSV export still works (exports flat process list, not groups)
