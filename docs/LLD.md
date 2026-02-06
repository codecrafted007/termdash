# Low-Level Design: termdash Interactive Process Monitor

## Table of Contents

1. [Module Specifications](#1-module-specifications)
2. [Data Structures](#2-data-structures)
3. [State Machine](#3-state-machine)
4. [Message & Command Protocol](#4-message--command-protocol)
5. [Metrics Layer Detail](#5-metrics-layer-detail)
6. [UI Model Detail](#6-ui-model-detail)
7. [Panel Renderer Specifications](#7-panel-renderer-specifications)
8. [Layout Engine](#8-layout-engine)
9. [Style System](#9-style-system)
10. [Keyboard Input Handling](#10-keyboard-input-handling)
11. [Scroll & Cursor Management](#11-scroll--cursor-management)
12. [Sort Implementation](#12-sort-implementation)
13. [History Ring Buffer](#13-history-ring-buffer)
14. [Query/Search System](#14-querysearch-system)
15. [Edge Cases & Boundary Conditions](#15-edge-cases--boundary-conditions)
16. [File-by-File Reference](#16-file-by-file-reference)

---

## 1. Module Specifications

### 1.1 internal/metrics — Data Collection

**Responsibility:** Collect system metrics from the OS via gopsutil. Provide structured data types. No UI knowledge.

**Exported API:**

```go
// Types
type CPUMetrics struct { PerCore []float64; Total float64 }
type MemoryMetrics struct { TotalRAM, UsedRAM uint64; RAMPercent float64; TotalSwap, UsedSwap uint64; SwapPercent float64 }
type DiskMetrics struct { MountPoint, Device string; Total, Used uint64; Percent float64 }
type NetworkMetrics struct { BytesSent, BytesRecv uint64; SendRate, RecvRate float64 }
type ProcessInfo struct { PID int32; Name string; CPUPercent float64; MemPercent float32; User string; MemRSS uint64; ConnCount int }
type Snapshot struct { Timestamp time.Time; CPU CPUMetrics; Memory MemoryMetrics; Disks []DiskMetrics; Network NetworkMetrics; Processes []ProcessInfo; Hostname, OS string; Uptime time.Duration }
type ProcessDetail struct { PID int32; Name, Cmdline, Exe, User string; CreateTime time.Time; CPUPercent float64; MemPercent float32; MemRSS, MemVMS uint64; NumThreads, NumFDs int32; ConnCount int; Environ []string; Status string; Nice, ParentPID int32; Err string }

// Collector
func NewCollector() *Collector
func (c *Collector) Collect() (Snapshot, error)

// Standalone functions
func CollectConnectionCounts() map[int32]int
func CollectProcessDetail(pid int32, connCount int) (*ProcessDetail, error)
func FormatBytes(b uint64) string
func FormatRate(bytesPerSec float64) string
```

**Dependency graph:**
```
collector.go → cpu.go, memory.go, disk.go, network.go, process.go
connections.go → gopsutil/net
detail.go → gopsutil/process
format.go → (none, pure functions)
```

### 1.2 internal/ui — Bubbletea Model

**Responsibility:** Application state machine. Handles all keyboard input, dispatches async commands, and composes the final view string.

**Exported API:**
```go
func New() Model
func NewHistory(capacity int) *History

// Model implements tea.Model:
func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

### 1.3 internal/ui/panels — Stateless Renderers

**Responsibility:** Pure functions that accept data and return styled strings. No state, no side effects.

**Exported API:**
```go
func RenderHeader(snap metrics.Snapshot) string
func RenderSummary(snap metrics.Snapshot, width int) string
func RenderProcessTable(processes []ProcessInfo, connCounts map[int32]int, cursor, scrollOffset, visibleRows int, sortCol SortColumn, sortAsc bool, width int) string
func RenderNetTop(processes []ProcessInfo, connCounts map[int32]int, snap Snapshot, width int) string
func RenderProcessDetail(detail *ProcessDetail, cpuHistory, memHistory []float64, width, height int) string
func RenderHelp(showHelp bool, inDetail bool) string
```

### 1.4 internal/ui/styles — Centralized Theme

**Responsibility:** Define all colors, lipgloss styles, and utility functions used across panels.

---

## 2. Data Structures

### 2.1 ProcessInfo (metrics/collector.go)

```go
type ProcessInfo struct {
    PID        int32    // OS process ID
    Name       string   // Short process name (e.g., "chrome")
    CPUPercent float64  // CPU usage 0.0–100.0+ (can exceed 100 on multi-core)
    MemPercent float32  // Memory usage as percentage of total RAM
    User       string   // Process owner username
    MemRSS     uint64   // Resident Set Size in bytes
    ConnCount  int      // Network connections; -1 = not yet populated
}
```

**ConnCount lifecycle:**
1. `collectProcesses()` sets `ConnCount = -1` (sentinel: "not yet known")
2. `collectConnCountsCmd()` produces `connCountMsg` with the global PID→count map
3. `sortedProcesses()` overlays `connCounts[PID]` onto each process before sorting
4. `RenderProcessTable()` checks both `connCounts[PID]` and `p.ConnCount >= 0` to decide display

### 2.2 ProcessDetail (metrics/detail.go)

```go
type ProcessDetail struct {
    PID        int32
    Name       string
    Cmdline    string      // Full command line with arguments
    Exe        string      // Absolute path to executable
    User       string
    CreateTime time.Time   // Process start time
    CPUPercent float64
    MemPercent float32
    MemRSS     uint64      // Resident Set Size
    MemVMS     uint64      // Virtual Memory Size
    NumThreads int32
    NumFDs     int32       // -1 if unavailable (macOS without root)
    ConnCount  int         // Passed in from connCounts map
    Environ    []string    // Truncated: max 50 vars, 200 chars each
    Status     string      // Process status code ("R", "S", "Z", etc.)
    Nice       int32       // Nice value (-20 to +19)
    ParentPID  int32
    Err        string      // Semicolon-separated per-field error list
}
```

**Error accumulation pattern:**
Each field is collected independently with its own error handling. Failed fields get default/sentinel values, and the error message is appended to `Err`. This ensures maximum data availability even when some fields are permission-denied.

### 2.3 Model (ui/model.go)

```go
type Model struct {
    // Core state
    collector   *metrics.Collector  // Stateful: tracks prev network bytes for rate calc
    snapshot    metrics.Snapshot    // Latest full system snapshot
    cpuHistory  *History            // 60-sample ring buffer for total CPU %
    sendHistory *History            // 60-sample ring buffer for network send rate
    recvHistory *History            // 60-sample ring buffer for network recv rate
    showHelp    bool                // Help overlay toggle
    width       int                 // Terminal width in columns
    height      int                 // Terminal height in rows
    err         error               // Last collection error (displayed to user)
    ready       bool                // True after first successful snapshot

    // Interactive state
    viewState     ViewState         // ViewDashboard | ViewProcessDetail
    cursor        int               // Index into sorted process list (0-based)
    scrollOffset  int               // First visible row in process table
    sortColumn    SortColumn        // Active sort column enum
    sortAscending bool              // Sort direction (default: descending)
    connCounts    map[int32]int     // PID → connection count (updated each cycle)

    // Detail view state (nil when not in detail view)
    detailPID        int32          // PID of process being inspected
    detailInfo       *metrics.ProcessDetail  // Latest detail snapshot
    detailCPUHistory *History       // 60-sample CPU% history for detail sparkline
    detailMemHistory *History       // 60-sample MEM% history for detail sparkline

    // Process grouping
    groupedView    bool             // Toggle with 'p' key
    expandedGroups map[string]bool  // Group name → expanded state

    // Query/search state
    queryMode    bool              // True when in query input mode
    queryType    QueryType         // QuerySearch (/) or QueryFilter (Q)
    queryInput   textinput.Model   // bubbles text input component
    queryFilter  *FilterNode       // Parsed filter AST (nil if invalid/empty)
    queryError   string            // Parse error message to display
    filterLocked bool              // True when filter is active but not editing
}
```

### 2.4 History Ring Buffer (ui/model.go)

```go
type History struct {
    data []float64  // Backing slice, capacity-limited
    cap  int        // Maximum number of elements
}
```

**Invariants:**
- `len(data) <= cap` always
- `Push()` evicts `data[0]` when at capacity (FIFO)
- `Values()` returns a defensive copy (mutations don't affect buffer)

### 2.5 Enums

```go
type ViewState int
const (
    ViewDashboard     ViewState = iota  // 0
    ViewProcessDetail                    // 1
)

type SortColumn int
const (
    SortByCPU   SortColumn = iota  // 0 (default)
    SortByMem                       // 1
    SortByPID                       // 2
    SortByName                      // 3
    SortByConns                     // 4
)
```

`SortColumn` is defined in both `internal/ui/model.go` and `internal/ui/panels/proctable.go` with identical iota values. The model casts between them: `panels.SortColumn(m.sortColumn)`. This avoids an import cycle.

### 2.6 Query Types (ui/query.go)

```go
// QueryType distinguishes simple search from field queries
type QueryType int
const (
    QuerySearch QueryType = iota  // "/" simple substring search
    QueryFilter                   // "Q" field-based DSL query
)

// FilterOp represents comparison operators
type FilterOp int
const (
    OpEq       FilterOp = iota  // =   equals
    OpNeq                        // !=  not equals
    OpGt                         // >   greater than
    OpLt                         // <   less than
    OpGte                        // >=  greater or equal
    OpLte                        // <=  less or equal
    OpContains                   // ~   contains (string substring)
)

// FilterExpr represents a single filter condition
type FilterExpr struct {
    Field string    // "name", "user", "pid", "cpu", "mem", "conn"
    Op    FilterOp  // comparison operator
    Value string    // value to compare against (parsed to type when evaluating)
}

// FilterNode represents a parsed query AST (supports AND/OR)
type FilterNode struct {
    Expr  *FilterExpr  // leaf node (non-nil for simple expressions)
    Op    string       // "and" or "or" for branch nodes
    Left  *FilterNode  // left subtree
    Right *FilterNode  // right subtree
}
```

**Query Grammar:**
```
query   := or_expr
or_expr := and_expr ("or" and_expr)*
and_expr:= primary ("and" primary)*
primary := field op value
field   := "name" | "user" | "pid" | "cpu" | "mem" | "conn"
op      := "=" | "!=" | ">" | "<" | ">=" | "<=" | "~"
value   := word | quoted_string
```

**Type coercion in evaluation:**
- `pid`, `conn` → parsed as `int64`
- `cpu`, `mem` → parsed as `float64`
- `name`, `user` → compared as lowercase strings

---

## 3. State Machine

### 3.1 View State Transitions

```
                 ┌─────────────────────┐
                 │   ViewDashboard     │◄────────────────────┐
                 │                     │                     │
                 │ Keys: j/k/g/G/s/S  │    Esc              │
                 │       ?/q/Ctrl+C    │                     │
                 └─────────┬───────────┘                     │
                           │                                 │
                    Enter (on process)                       │
                           │                                 │
                           ▼                                 │
                 ┌─────────────────────┐                     │
                 │ ViewProcessDetail   │─────────────────────┘
                 │                     │  also: process exits
                 │ Keys: Esc/?/q/      │  (processDetailMsg.err != nil)
                 │       Ctrl+C        │
                 └─────────────────────┘
```

### 3.2 Message Flow State Machine

```
State: INIT
  │
  └─→ Init() → Batch(tickCmd, collectCmd)
         │              │
         ▼              ▼
     tickMsg        snapshotMsg
         │              │
         │              ├─→ Store snapshot, push histories
         │              ├─→ Clamp cursor
         │              └─→ Batch(tickCmd, collectConnCountsCmd)
         │                              │
         └──→ collectCmd()              ▼
                                   connCountMsg
                                        │
                                        └─→ Store in model.connCounts

State: DASHBOARD (viewState == ViewDashboard)
  ├─ KeyMsg "j"/"k" → adjust cursor, ensure visible
  ├─ KeyMsg "g"/"G" → jump to top/bottom
  ├─ KeyMsg "s"     → cycle sortColumn (0→1→2→3→4→0), reset ascending=false
  ├─ KeyMsg "S"     → toggle sortAscending
  ├─ KeyMsg "enter" → transition to DETAIL
  ├─ KeyMsg "/"     → transition to QUERY (QuerySearch mode)
  ├─ KeyMsg "Q"     → transition to QUERY (QueryFilter mode)
  ├─ KeyMsg "p"     → toggle process grouping
  ├─ KeyMsg "?"     → toggle showHelp
  └─ KeyMsg "q"     → tea.Quit

State: QUERY (queryMode == true)
  ├─ On entry:
  │   ├─ queryMode = true
  │   ├─ queryType = QuerySearch or QueryFilter
  │   ├─ queryInput.SetValue(""), queryInput.Focus()
  │   └─ Return textinput.Blink command
  │
  ├─ KeyMsg (any character) → update queryInput, applyQueryFilter()
  │   ├─ Parse query (DSL mode) or use as substring (search mode)
  │   ├─ On parse error: queryError = error message, queryFilter = nil
  │   ├─ On success: queryError = "", queryFilter = parsed AST
  │   └─ Clamp cursor to filtered results
  │
  ├─ KeyMsg "enter" → lock filter, exit query mode
  │   ├─ filterLocked = true (filter remains active)
  │   ├─ queryMode = false
  │   └─ statusMsg = "Filter: <query>"
  │
  ├─ KeyMsg "esc"   → clear/exit query mode
  │   ├─ If filterLocked: clear filter, unlock
  │   ├─ queryMode = false
  │   └─ Return to unfiltered view
  │
  └─ KeyMsg "ctrl+c" → tea.Quit (even in query mode)

State: DETAIL (viewState == ViewProcessDetail)
  ├─ On entry:
  │   ├─ Create detailCPUHistory, detailMemHistory
  │   └─ Dispatch: Batch(collectDetailCmd, detailTickCmd)
  │
  ├─ processDetailMsg (success) → store detailInfo, push to histories
  ├─ processDetailMsg (error)   → transition to DASHBOARD (process exited)
  ├─ detailTickMsg → Batch(collectDetailCmd, detailTickCmd) [1s loop]
  ├─ detailTickMsg (if viewState != DETAIL) → noop (guard clause)
  ├─ KeyMsg "esc"   → transition to DASHBOARD, nil detail state
  ├─ KeyMsg "?"     → toggle showHelp
  └─ KeyMsg "q"     → tea.Quit
```

---

## 4. Message & Command Protocol

### 4.1 Message Types

| Message Type | Payload | Sender | Handler |
|-------------|---------|--------|---------|
| `tickMsg` | `time.Time` | `tickCmd()` (2s timer) | Dispatches `collectCmd()` |
| `snapshotMsg` | `metrics.Snapshot` | `collectCmd()` | Stores snapshot, pushes histories, dispatches tick + connCounts |
| `connCountMsg` | `map[int32]int` | `collectConnCountsCmd()` | Stores in `model.connCounts` |
| `processDetailMsg` | `{detail *ProcessDetail, err error}` | `collectDetailCmd()` | Stores detail, pushes detail histories; on error → return to dashboard |
| `detailTickMsg` | `time.Time` | `detailTickCmd()` (1s timer) | Dispatches `collectDetailCmd` + `detailTickCmd` (if still in detail view) |
| `errMsg` | `error` | `collectCmd()` on failure | Stores error, displays to user, continues tick |
| `tea.KeyMsg` | `string` (key name) | Bubbletea runtime | Routed through `handleKey()` → `handleDashboardKey()` or `handleDetailKey()` |
| `tea.WindowSizeMsg` | `{Width, Height int}` | Bubbletea runtime | Updates `model.width` and `model.height` |

### 4.2 Command Definitions

```go
// tickCmd: fire-and-forget timer, returns tickMsg after 2s
func tickCmd() tea.Cmd {
    return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// collectCmd: run Collector.Collect() in goroutine
func collectCmd(c *metrics.Collector) tea.Cmd {
    return func() tea.Msg {
        snap, err := c.Collect()
        if err != nil { return errMsg(err) }
        return snapshotMsg(snap)
    }
}

// collectConnCountsCmd: run CollectConnectionCounts() in goroutine
func collectConnCountsCmd() tea.Cmd {
    return func() tea.Msg { return connCountMsg(metrics.CollectConnectionCounts()) }
}

// collectDetailCmd: run CollectProcessDetail() in goroutine
func collectDetailCmd(pid int32, connCount int) tea.Cmd {
    return func() tea.Msg {
        detail, err := metrics.CollectProcessDetail(pid, connCount)
        return processDetailMsg{detail: detail, err: err}
    }
}

// detailTickCmd: fire-and-forget timer, returns detailTickMsg after 1s
func detailTickCmd() tea.Cmd {
    return tea.Tick(1*time.Second, func(t time.Time) tea.Msg { return detailTickMsg(t) })
}
```

### 4.3 Command Dispatch Patterns

| Trigger | Commands Dispatched | Pattern |
|---------|-------------------|---------|
| `Init()` | `tickCmd` + `collectCmd` | `tea.Batch()` — parallel |
| `snapshotMsg` received | `tickCmd` + `collectConnCountsCmd` | `tea.Batch()` — parallel |
| `Enter` key (dashboard) | `collectDetailCmd` + `detailTickCmd` | `tea.Batch()` — parallel |
| `detailTickMsg` (in detail) | `collectDetailCmd` + `detailTickCmd` | `tea.Batch()` — parallel |
| `tickMsg` received | `collectCmd` | Single command |

---

## 5. Metrics Layer Detail

### 5.1 Collector.Collect() — Orchestration

```
Collect()
├── collectCPU()        → CPUMetrics      [gopsutil/cpu.Percent]
├── collectMemory()     → MemoryMetrics   [gopsutil/mem.VirtualMemory + SwapMemory]
├── collectDisks()      → []DiskMetrics   [gopsutil/disk.Partitions + Usage]
├── collectNetwork(now) → NetworkMetrics  [gopsutil/net.IOCounters + rate calc]
├── collectProcesses(25)→ []ProcessInfo   [gopsutil/process.Processes + sort + truncate]
└── collectHostInfo()   → hostname, OS, uptime [gopsutil/host.Info]
```

**Rate calculation (network.go):**
```
sendRate = (currentBytesSent - prevBytesSent) / elapsedSeconds
recvRate = (currentBytesRecv - prevBytesRecv) / elapsedSeconds
```
First call returns rate=0 (no previous baseline). `Collector` stores `prevBytesSent`, `prevBytesRecv`, `prevTime` for delta calculation.

### 5.2 collectProcesses(topN=25)

```
1. process.Processes()                → all OS processes
2. For each process:
   ├── p.Name()          → skip on error
   ├── p.CPUPercent()    → skip on error
   ├── p.MemoryPercent() → skip on error
   ├── p.Username()      → "" on error (non-fatal)
   └── p.MemoryInfo()    → RSS; 0 on error (non-fatal)
3. Sort by CPUPercent descending
4. Truncate to top 25
5. Set ConnCount = -1 for all (populated later by connection collector)
```

### 5.3 CollectConnectionCounts()

```
1. psnet.Connections("all")  → []ConnectionStat
   ├── On error → return empty map (graceful degradation)
2. For each connection where PID > 0:
   └── counts[PID]++
3. Return counts map
```

**Syscall behavior:** On macOS, this reads from `lsof`-style system calls. On Linux, it reads `/proc/net/tcp`, `/proc/net/tcp6`, `/proc/net/udp`, `/proc/net/udp6`. May require root on some configurations.

### 5.4 CollectProcessDetail(pid, connCount)

```
1. process.NewProcess(pid)    → error if process doesn't exist
2. Collect each field independently:
   ├── p.Name()        → d.Name       or append to errs
   ├── p.Cmdline()     → d.Cmdline    or append to errs
   ├── p.Exe()         → d.Exe        or append to errs
   ├── p.Username()    → d.User       or append to errs
   ├── p.CreateTime()  → d.CreateTime or append to errs
   ├── p.CPUPercent()  → d.CPUPercent or append to errs
   ├── p.MemoryPercent() → d.MemPercent or append to errs
   ├── p.MemoryInfo()  → d.MemRSS, d.MemVMS or append to errs
   ├── p.NumThreads()  → d.NumThreads or append to errs
   ├── p.NumFDs()      → d.NumFDs; -1 on error (macOS common)
   ├── p.Environ()     → d.Environ (truncated) or append to errs
   ├── p.Status()      → d.Status; "?" on error
   ├── p.Nice()        → d.Nice or append to errs
   └── p.Ppid()        → d.ParentPID or append to errs
3. d.ConnCount = connCount (passed in, not re-collected)
4. d.Err = join(errs, "; ") if any errors
5. Return d, nil (error only if process doesn't exist at all)
```

**Environ truncation:**
- Max 50 environment variables
- Each variable truncated to 200 characters + "..." suffix
- Prevents memory exhaustion from pathological environments

### 5.5 Disk Filtering (disk.go)

Pseudo-filesystems are excluded by `Fstype` prefix:
- `devfs` (macOS device filesystem)
- `tmpfs` (Linux temp filesystem)
- `proc` (Linux proc filesystem)
- `sysfs` (Linux sys filesystem)

Mount points are deduplicated by a `seen` map. Zero-total partitions are excluded.

---

## 6. UI Model Detail

### 6.1 Initialization

```go
func New() Model {
    return Model{
        collector:   metrics.NewCollector(),
        cpuHistory:  NewHistory(60),
        sendHistory: NewHistory(60),
        recvHistory: NewHistory(60),
        width:       80,     // default; overridden by first WindowSizeMsg
        height:      24,     // default; overridden by first WindowSizeMsg
        connCounts:  map[int32]int{},
    }
}
```

Default dimensions (80x24) are used only until the first `WindowSizeMsg` arrives from Bubbletea (typically within the first frame).

### 6.2 Update() — Message Router

```
Update(msg)
├── tea.KeyMsg       → handleKey(msg)
│                       ├── "q" / "ctrl+c" → tea.Quit
│                       ├── "?"            → toggle showHelp
│                       ├── ViewDashboard  → handleDashboardKey()
│                       └── ViewDetail     → handleDetailKey()
│
├── tea.WindowSizeMsg → store width, height
│
├── tickMsg          → collectCmd(collector)
│
├── snapshotMsg      → store snapshot
│                       push to 3 histories
│                       clamp cursor
│                       Batch(tickCmd, collectConnCountsCmd)
│
├── connCountMsg     → store map in model.connCounts
│
├── processDetailMsg → if error: return to dashboard
│                       else: store detail, push to detail histories
│
├── detailTickMsg    → if not in detail view: noop
│                       else: Batch(collectDetailCmd, detailTickCmd)
│
└── errMsg           → store error, tickCmd (retry next cycle)
```

### 6.3 View() — Render Dispatch

```
View()
├── !ready          → "Loading system metrics..."
├── err != nil      → "Error: ..."
├── ViewDetail      → renderDetailView()
└── ViewDashboard   → renderDashboardView()

renderDashboardView():
  1. RenderHeader(snapshot)
  2. RenderSummary(snapshot, width)
  3. sortedProcesses() → sorted, conn-annotated copy
  4. Calculate visibleRows = height - 11
  5. If visibleRows < 5: collapse netTop, reclaim 2 rows
  6. RenderProcessTable(procs, connCounts, cursor, scroll, visible, sortCol, sortAsc, width)
  7. RenderNetTop(procs, connCounts, snapshot, width) — if not collapsed
  8. RenderHelp(showHelp, false)
  9. RenderDashboardLayout(width, height, header, summary, table, netTop, help)

renderDetailView():
  1. Get cpuHistory, memHistory from detail buffers (or nil)
  2. RenderProcessDetail(detailInfo, cpuHist, memHist, width, height)
  3. RenderHelp(showHelp, true)
  4. RenderDetailLayout(width, height, detailView, help)
```

---

## 7. Panel Renderer Specifications

### 7.1 RenderHeader (panels/header.go)

**Input:** `metrics.Snapshot`
**Output:** Single-line string

**Layout:**
```
⣿ termdash  Host: hostname  OS: darwin 15.0  Uptime: 5d 3h 12m  Time: 14:32:05
```

**Uptime formatting:**
- `>= 1 day`: `"Xd Yh Zm"`
- `< 1 day`: `"Yh Zm"`

### 7.2 RenderSummary (panels/summary.go)

**Input:** `metrics.Snapshot`, `width int`
**Output:** 3-line string (CPU, MEM, Swap)

**Bar width selection:**
- `width >= 60`: barWidth = 20
- `width < 60`: barWidth = 10

**CPU line construction:**
```
"CPU [████████░░░░░░░░░░░░] 32%"  +  per-core suffixes until width exceeded
                                      e.g., "  0:45%  1:28%  2:35%"
```
Per-core suffixes are added greedily: each `"  N:XX%"` appended if `len(line) + len(suffix) <= width - 2`.

**Bar character rendering:**
```
filled = int(pct / 100.0 * barWidth)  — clamped to [0, barWidth]
empty  = barWidth - filled
output = "[" + "█"×filled + "░"×empty + "]"
```

### 7.3 RenderProcessTable (panels/proctable.go)

**Input:** sorted processes, connCounts, cursor, scrollOffset, visibleRows, sortCol, sortAsc, width
**Output:** Multi-line string with header, separator, data rows, scroll indicators

**Column layout:**

| Column | Width | Alignment | Sort indicator |
|--------|-------|-----------|----------------|
| PID | 7 | left | `PID▼` / `PID▲` when active |
| NAME | fill (remaining) | left | `NAME▼` / `NAME▲` when active |
| USER | 10 | left | (not sortable) |
| CPU% | 6 | right | `CPU%▼` / `CPU%▲` when active |
| MEM% | 6 | right | `MEM%▼` / `MEM%▲` when active |
| RSS | 8 | right | (not sortable) |
| CONN | 5 | right | `CONN▼` / `CONN▲` when active |

`nameW = innerWidth - 7 - 10 - 6 - 6 - 8 - 5 - 9(gaps)` — minimum 6.

**Scroll indicators:**
- `hasAbove = scrollOffset > 0` → render `"▲ more"` and decrement visibleRows
- `hasBelow = scrollOffset + visibleRows < len(processes)` → reserve 1 row for `"▼ more"`

**Cursor rendering:**
- Non-selected row prefix: `"  "`
- Selected row prefix: `"> "`
- Selected row styled with `CursorRowStyle` (purple background, white text)

**CONN column display logic:**
```
if connCounts[PID] exists → show count
else if p.ConnCount >= 0  → show p.ConnCount
else                      → show "-"
```

**Truncation:**
- `Name` > `nameW`: truncated to `nameW-1` + "..."
- `User` > `userW`: truncated to `userW-1` + "..."
- `RSS` string > `rssW`: truncated to `rssW`

### 7.4 RenderNetTop (panels/nettop.go)

**Input:** sorted processes, connCounts, snapshot, width
**Output:** Single-line styled string

**Construction:**
1. Format rate string: `"Net: ▲{sendRate} ▼{recvRate}"`
2. Aggregate connections by process name (sum PIDs with same name)
3. Sort by count descending, take top 5
4. Format: `" Top: chrome(15) node(12) code(8)"`
5. Truncate top string if `len(rateStr) + len(sep) + len(topStr) + 2 > width`
6. Apply color styles: send=red arrows, recv=green arrows, top=orange

### 7.5 RenderProcessDetail (panels/procdetail.go)

**Input:** `*ProcessDetail` (may be nil), cpuHistory, memHistory `[]float64`, width, height
**Output:** Full detail view string

**Nil detail handling:** Returns `"Loading process details..."` with label style.

**Section layout:**

```
Section 1: Title bar (DetailHeaderStyle)
  "Process: {name} (PID {pid})  ── [Esc] back"

Section 2-3: Info lines (LabelStyle)
  "User: {user}      Status: {status}  Threads: {threads}  FDs: {fds|N/A}"
  "Parent: {ppid}    Nice: {nice}      Conns: {conns}      Created: {time}"

Section 4: Command line
  "Cmd: " (LabelStyle) + "{cmdline|exe}" (ValueStyle)
  Truncated to innerWidth-6 chars, with "..." suffix

Section 5: Separator
  "─" × innerWidth

Section 6: Side-by-side sparkline charts
  chartWidth = (innerWidth - 3) / 2    — min 10
  chartHeight = 3 rows
  Left:  CPU History sparkline + "Current: {cpu}%"
  Right: Memory History sparkline + "RSS: {rss}  VMS: {vms}"
  Joined horizontally with " │ " separator

Section 7: Separator

Section 8: Environment variables
  "Environment:" header
  If empty: "(not available — may require elevated privileges)"
  If present: show envLines = height - sections_so_far - 3
    Each line: "  {VAR=value}" truncated to innerWidth-3 + "..."
    Overflow: "  ... and {N} more"
```

**Sparkline rendering (renderDetailSparkline):**
```
if data empty → show "░" × width placeholder
else → ntcharts sparkline with maxValue=100, colored by metric type
```

### 7.6 RenderHelp (panels/help.go)

**Input:** `showHelp bool`, `inDetail bool`
**Output:** Single-line string

**Compact mode (`showHelp=false`):**
- Dashboard: `" j/k:move  Enter:detail  s:sort  ?:help  q:quit"`
- Detail: `" Esc:back  ?:help  q:quit"`

**Expanded mode (`showHelp=true`):**
- Dashboard: 7 keybindings with styled key + description
- Detail: 4 keybindings with styled key + description

---

## 8. Layout Engine

### 8.1 RenderDashboardLayout (layout.go)

**Input:** width, height, header, summary, processTable, netTop, help — all pre-rendered strings
**Output:** Vertically stacked composition

```go
parts = [header(width-constrained), summary(width-constrained), processTable]
if netTop != "" { parts = append(parts, netTop) }
parts = append(parts, help(width-constrained))
return lipgloss.JoinVertical(lipgloss.Left, parts...)
```

Minimum width enforced: 40 columns.

### 8.2 RenderDetailLayout (layout.go)

```go
return lipgloss.JoinVertical(lipgloss.Left, detailView, help(width-constrained))
```

### 8.3 Visible Row Calculation

```
visibleRows = termHeight - overhead
overhead = 11 (header:1 + summary:3 + tableHeader:2 + netTop:2 + help:1 + padding:2)
minimum = 1

If visibleRows < 5:
  - netTop is collapsed (not rendered)
  - visibleRows += 2 (reclaimed from netTop)
```

### 8.4 Legacy Layout (RenderLayout)

Preserved for backward compatibility with old panel renderers. Uses 2-column layout when `width >= 80` (left: CPU+Memory, right: Disk+Network), single-column otherwise.

---

## 9. Style System

### 9.1 Color Palette

| Name | Hex | Usage |
|------|-----|-------|
| `ColorPrimary` | `#7D56F4` | Purple — titles, sort indicators, help keys |
| `ColorSecondary` | `#6C63FF` | Indigo — disk bars |
| `ColorSuccess` | `#04B575` | Green — normal usage, RAM, recv |
| `ColorWarning` | `#FFBE0B` | Amber — medium usage, swap |
| `ColorDanger` | `#FF6B6B` | Red — high usage, send |
| `ColorMuted` | `#626262` | Dim gray — borders, empty bars, help text |
| `ColorText` | `#FAFAFA` | Near-white — primary text |
| `ColorSubtle` | `#A0A0A0` | Light gray — labels, secondary text |
| `ColorConn` | `#FF8C00` | Orange — connection counts |

### 9.2 Dynamic Color Function

```go
func UsageColor(pct float64) lipgloss.Color {
    >= 90% → ColorDanger  (#FF6B6B)
    >= 70% → ColorWarning (#FFBE0B)
    <  70% → ColorSuccess (#04B575)
}
```

### 9.3 Composite Styles

| Style | Properties | Usage |
|-------|-----------|-------|
| `PanelStyle` | Rounded border, muted foreground, padding(0,1) | Legacy panel wrapping |
| `HeaderTitleStyle` | Bold, primary foreground | App title "termdash" |
| `SectionTitleStyle` | Bold, text foreground, marginBottom(1) | Panel section headers |
| `LabelStyle` | Subtle foreground | Metric labels, info text |
| `ValueStyle` | Text foreground, bold | Metric values |
| `HelpKeyStyle` | Primary foreground, bold | Keyboard shortcut labels |
| `HelpDescStyle` | Muted foreground | Help descriptions |
| `CursorRowStyle` | Background `#3A3A5C`, text foreground | Selected process row |
| `SortIndicatorStyle` | Bold, primary foreground | Sort arrows on column headers |
| `DetailHeaderStyle` | Bold, text foreground, bg `#3A3A5C`, padding(0,1) | Detail view title bar |

---

## 10. Keyboard Input Handling

### 10.1 Input Routing

```
tea.KeyMsg → handleKey()
  ├── If queryMode == true → handleQueryInput()
  │   ├── "esc"    → clear filter if locked, exit query mode
  │   ├── "enter"  → lock filter, exit query mode
  │   ├── "ctrl+c" → tea.Quit
  │   └── (other)  → pass to textinput, applyQueryFilter()
  │
  ├── Global keys (all views):
  │   ├── "q" / "ctrl+c" → tea.Quit
  │   └── "?"            → toggle showHelp
  │
  ├── ViewDashboard → handleDashboardKey()
  │   ├── "j" / "down"  → cursor++, ensureCursorVisible()
  │   ├── "k" / "up"    → cursor--, ensureCursorVisible()
  │   ├── "g" / "home"  → cursor=0, scrollOffset=0
  │   ├── "G" / "end"   → cursor=last, ensureCursorVisible()
  │   ├── "enter"       → enter detail view for selected process
  │   ├── "s"           → sortColumn = (sortColumn+1) % 5, ascending=false
  │   ├── "S"           → toggle sortAscending
  │   ├── "p"           → toggle process grouping
  │   ├── "/"           → enter query mode (QuerySearch)
  │   └── "Q"           → enter query mode (QueryFilter)
  │
  └── ViewProcessDetail → handleDetailKey()
      └── "esc"         → return to dashboard, clear detail state
```

### 10.2 Key Binding Summary

| Key | View | Action |
|-----|------|--------|
| `j` / `Down` | Dashboard | Move cursor down |
| `k` / `Up` | Dashboard | Move cursor up |
| `g` / `Home` | Dashboard | Jump to first process |
| `G` / `End` | Dashboard | Jump to last process |
| `Enter` | Dashboard | Open detail view for selected process |
| `s` | Dashboard | Cycle sort column: CPU% → MEM% → PID → NAME → CONN → CPU% |
| `S` | Dashboard | Toggle sort direction (ascending/descending) |
| `p` | Dashboard | Toggle process grouping view |
| `/` | Dashboard | Enter simple search mode (by name/PID) |
| `Q` | Dashboard | Enter query filter mode (DSL) |
| `Enter` | Query Mode | Lock current filter and exit query mode |
| `Esc` | Query Mode | Clear filter (if locked) and exit query mode |
| `Esc` | Detail | Return to dashboard |
| `?` | Both | Toggle expanded help |
| `q` | Both | Quit application |
| `Ctrl+C` | Both | Force quit |

---

## 11. Scroll & Cursor Management

### 11.1 ensureCursorVisible()

```go
func (m *Model) ensureCursorVisible() {
    visible := m.visibleRows()  // dynamic based on terminal height
    if visible <= 0 { visible = 1 }

    // Scroll up if cursor is above viewport
    if m.cursor < m.scrollOffset {
        m.scrollOffset = m.cursor
    }

    // Scroll down if cursor is below viewport
    if m.cursor >= m.scrollOffset + visible {
        m.scrollOffset = m.cursor - visible + 1
    }
}
```

**Note:** This uses a pointer receiver (`*Model`) because it's called from value-receiver methods (`handleDashboardKey`). Go allows calling pointer methods on addressable local variables. The local `m` copy is modified and then returned.

### 11.2 Cursor Clamping on Snapshot Update

```go
// In snapshotMsg handler:
if m.cursor >= len(m.snapshot.Processes) && len(m.snapshot.Processes) > 0 {
    m.cursor = len(m.snapshot.Processes) - 1
}
```

This handles the case where a process exits between snapshots and the cursor would point past the end of the list.

### 11.3 Scroll Indicator Logic (in RenderProcessTable)

```
hasAbove = scrollOffset > 0
  → render "▲ more" at top, consume 1 visibleRow

hasBelow = scrollOffset + visibleRows < len(processes)
  → render "▼ more" at bottom, reserve 1 visibleRow

Actual data rows rendered: from scrollOffset to min(scrollOffset+adjusted_visibleRows, len(processes))
```

---

## 12. Sort Implementation

### 12.1 Sort at Render Time

The collector always returns processes sorted by CPU% descending. User sort preferences are applied in `sortedProcesses()`, called by `View()` methods.

```go
func (m Model) sortedProcesses() []metrics.ProcessInfo {
    // 1. Create defensive copy of process list
    procs := make([]metrics.ProcessInfo, len(m.snapshot.Processes))
    copy(procs, m.snapshot.Processes)

    // 2. Overlay connection counts from connCounts map
    for i := range procs {
        if c, ok := m.connCounts[procs[i].PID]; ok {
            procs[i].ConnCount = c
        }
    }

    // 3. Define comparator based on sortColumn
    less := func(i, j int) bool {
        switch m.sortColumn {
        case SortByCPU:   return procs[i].CPUPercent > procs[j].CPUPercent
        case SortByMem:   return procs[i].MemPercent > procs[j].MemPercent
        case SortByPID:   return procs[i].PID < procs[j].PID
        case SortByName:  return procs[i].Name < procs[j].Name
        case SortByConns: return procs[i].ConnCount > procs[j].ConnCount
        default:          return procs[i].CPUPercent > procs[j].CPUPercent
        }
    }

    // 4. Apply direction: ascending inverts the comparator
    if m.sortAscending {
        sort.Slice(procs, func(i, j int) bool { return !less(i, j) })
    } else {
        sort.Slice(procs, less)
    }

    return procs
}
```

### 12.2 Default Sort Directions

| Column | Default (descending) | Meaning |
|--------|---------------------|---------|
| CPU% | Highest first | Most CPU-hungry processes on top |
| MEM% | Highest first | Most memory-hungry processes on top |
| PID | Lowest first | Oldest processes on top (ascending PID) |
| NAME | A→Z | Alphabetical |
| CONN | Highest first | Most connections on top |

Pressing `s` cycles column and resets direction to default (descending). Pressing `S` toggles ascending/descending for the current column.

### 12.3 Sort Header Display

```go
func sortHeader(label string, col, activeCol SortColumn, asc bool) string {
    if col == activeCol {
        arrow := "▼"      // descending
        if asc { arrow = "▲" }  // ascending
        return SortIndicatorStyle.Render(label + arrow)
    }
    return label  // inactive column, no arrow
}
```

---

## 13. History Ring Buffer

### 13.1 Implementation

```go
type History struct {
    data []float64
    cap  int
}

func NewHistory(capacity int) *History {
    return &History{data: make([]float64, 0, capacity), cap: capacity}
}

func (h *History) Push(v float64) {
    if len(h.data) >= h.cap {
        h.data = h.data[1:]   // evict oldest (index 0)
    }
    h.data = append(h.data, v)  // append newest at end
}

func (h *History) Values() []float64 {
    out := make([]float64, len(h.data))
    copy(out, h.data)
    return out  // defensive copy
}
```

### 13.2 Buffer Allocation

| Buffer | Capacity | Refresh | Lifetime | Data |
|--------|----------|---------|----------|------|
| `cpuHistory` | 60 | 2s (tick) | App lifetime | Total CPU % |
| `sendHistory` | 60 | 2s (tick) | App lifetime | Network send rate (bytes/s) |
| `recvHistory` | 60 | 2s (tick) | App lifetime | Network recv rate (bytes/s) |
| `detailCPUHistory` | 60 | 1s (detail tick) | Detail view only | Selected process CPU % |
| `detailMemHistory` | 60 | 1s (detail tick) | Detail view only | Selected process MEM % |

At 2-second intervals with capacity 60, the main histories represent ~2 minutes of data. Detail histories at 1-second intervals represent ~1 minute.

### 13.3 Memory Overhead

```
Per buffer: 60 × 8 bytes (float64) = 480 bytes
3 main buffers: 1,440 bytes
2 detail buffers (when active): 960 bytes
Total maximum: 2,400 bytes (~2.3 KiB)
```

---

## 14. Query/Search System

### 14.1 Overview

The query/search system provides two modes for filtering processes:

1. **Simple Search (`/`)**: Case-insensitive substring matching against process name or exact PID match
2. **Query Filter (`Q`)**: Field-based DSL supporting comparisons, AND/OR operators

### 14.2 Query DSL Specification

**Supported Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Process name |
| `user` | string | Username |
| `pid` | int | Process ID |
| `cpu` | float | CPU percentage |
| `mem` | float | Memory percentage |
| `conn` | int | Connection count |

**Operators:**

| Op | Meaning | Example |
|----|---------|---------|
| `=` | equals | `user = root` |
| `!=` | not equals | `name != chrome` |
| `>` | greater than | `cpu > 50` |
| `<` | less than | `mem < 10` |
| `>=` | greater or equal | `conn >= 5` |
| `<=` | less or equal | `cpu <= 1` |
| `~` | contains (strings) | `name ~ java` |

**Logical Operators:**
- `and` — binds tighter than `or`
- `or` — evaluated left-to-right after `and`

**Example Queries:**
```
name ~ nginx                    # Processes containing "nginx"
cpu > 50                        # High CPU processes
user = root and conn > 0        # Root processes with connections
name ~ java or name ~ python    # Java or Python processes
mem > 5 and cpu > 10            # Memory and CPU hogs
```

### 14.3 Parser Implementation (query.go)

**Tokenizer:**
```go
func tokenize(input string) ([]string, error)
```
- Handles quoted strings (`"my process"`)
- Recognizes multi-char operators (`!=`, `>=`, `<=`)
- Splits on whitespace
- Returns error on unterminated quotes

**Recursive Descent Parser:**
```go
func ParseQuery(input string) (*FilterNode, error)
```
- Entry point: `parseOr()` handles OR expressions
- `parseAnd()` handles AND expressions (higher precedence)
- `parsePrimary()` handles `field op value` atoms
- Validates field names (rejects unknown fields)
- Returns nil, nil for empty input

**AST Evaluation:**
```go
func (f *FilterNode) MatchProcess(p metrics.ProcessInfo, connCount int) bool
```
- Recursively evaluates AND/OR nodes
- Leaf nodes delegate to `FilterExpr.evaluate()`
- Type-specific comparison methods:
  - `compareString()` — case-insensitive string ops
  - `compareInt()` — integer comparisons (pid, conn)
  - `compareFloat()` — float comparisons (cpu, mem)

### 14.4 Simple Search Implementation

```go
func SimpleSearch(p metrics.ProcessInfo, query string) bool
```
- Returns true for empty query
- Case-insensitive substring match on `p.Name`
- Exact match on PID if query is numeric
- Substring match on PID string representation

### 14.5 Filter Integration (model.go)

**State Management:**
```go
queryMode    bool            // true when text input is focused
queryType    QueryType       // QuerySearch or QueryFilter
queryInput   textinput.Model // bubbles component
queryFilter  *FilterNode     // parsed AST (nil if invalid)
queryError   string          // parse error message
filterLocked bool            // true when filter active but not editing
```

**Key Methods:**

`handleQueryInput(msg tea.KeyMsg)`:
- Routes Esc/Enter specially
- Passes other keys to textinput
- Calls `applyQueryFilter()` on each keystroke

`applyQueryFilter()`:
- Parses current input (DSL mode only)
- Updates queryFilter and queryError
- Clamps cursor to filtered result count

`filteredProcesses() []ProcessInfo`:
- Wraps `sortedProcesses()`
- Applies filter via `SimpleSearch()` or `queryFilter.MatchProcess()`
- Returns filtered slice

`hasActiveFilter() bool`:
- Returns true if query input non-empty AND (queryMode OR filterLocked)

### 14.6 Query Bar Renderer (panels/querybar.go)

```go
func RenderQueryBar(queryType QueryType, input textinput.Model,
                    errorMsg string, filterLocked bool, width int) string
```

**Display States:**

1. **Active input**: `"Query: " + input.View() + errorMsg`
2. **Locked filter**: `"Query: " + value + " [locked - Esc to clear]"`
3. **Search mode**: Uses `"/"` prompt instead of `"Query: "`

**Styling:**
- Prompt: bold, primary color (purple)
- Background: dark (#2A2A3C)
- Error: danger color (red), italic
- Locked indicator: success color (green), italic

### 14.7 Layout Integration

Query bar is inserted between process table and netTop:

```go
// In renderDashboardView():
if m.queryMode || m.filterLocked {
    queryBar = panels.RenderQueryBar(...)
    visible-- // account for query bar height
}

// In RenderDashboardLayout():
parts := []string{header, summary, processTable}
if queryBar != "" {
    parts = append(parts, queryBar)
}
if netTop != "" {
    parts = append(parts, netTop)
}
parts = append(parts, help)
```

---

## 15. Edge Cases & Boundary Conditions

### 15.1 Terminal Size

| Condition | Handling |
|-----------|----------|
| Width < 40 | Clamped to 40 in layout functions |
| Width < 20 | Summary bar width set to minimum (barWidth=10) |
| Height produces `visibleRows < 5` | NetTop panel collapsed, 2 rows reclaimed |
| Height produces `visibleRows < 1` | Clamped to 1 |
| Width < 60 | Summary bar width reduced from 20 to 10 |
| `nameW < 6` after column math | Clamped to 6 |

### 15.2 Process Lifecycle

| Condition | Handling |
|-----------|----------|
| Process exits between snapshots | Cursor clamped: `if cursor >= len(procs) → cursor = len(procs)-1` |
| Process exits during detail view | `processDetailMsg.err != nil` → auto-return to dashboard |
| Process list empty | No rows rendered; `RenderProcessTable` produces header + separator only |
| New process appears | Picked up in next `collectProcesses(25)` call |

### 15.3 Permission Errors

| Condition | Handling |
|-----------|----------|
| `net.Connections("all")` fails (no root) | Returns empty map; CONN column shows "-" for all processes |
| `process.NumFDs()` fails (macOS sandbox) | `NumFDs = -1`; detail view shows "N/A" |
| `process.Environ()` fails (permissions) | Empty list; shows "(not available — may require elevated privileges)" |
| Individual detail field fails | Error appended to `ProcessDetail.Err`; other fields still populated |
| `Collect()` fails entirely | `errMsg` displayed; tick continues; next cycle retries |

### 15.4 Data Truncation

| Data | Limit | Handling |
|------|-------|---------|
| Process name | `nameW` chars | Truncated with "..." suffix |
| Username | 10 chars | Truncated with "..." suffix |
| RSS string | 8 chars | Truncated to fit |
| Cmdline (detail) | `innerWidth - 6` chars | Truncated with "..." suffix |
| Environment variable | 200 chars | Truncated with "..." suffix |
| Environment count | 50 variables max | Excess dropped silently |
| NetTop top procs | 5 processes max | Rest dropped |
| NetTop string | Available width | Truncated with "..." suffix |

### 15.5 Numeric Boundaries

| Value | Range | Notes |
|-------|-------|-------|
| CPU% per-process | 0.0 – N×100 | Can exceed 100% on multi-core systems |
| MEM% per-process | 0.0 – 100.0 | `float32` from gopsutil |
| ConnCount | -1 or 0+ | -1 sentinel means "not yet populated" |
| NumFDs | -1 or 0+ | -1 sentinel means "unavailable" |
| Nice | -20 to +19 | Standard Unix nice range |
| SortColumn | 0–4 | Modular arithmetic: `(col+1) % 5` |

### 15.6 Query/Search Edge Cases

| Condition | Handling |
|-----------|----------|
| Empty query | Show all processes (no filtering) |
| Query matches nothing | Show empty table (no crash, cursor clamped to 0) |
| Invalid DSL syntax | Show error in query bar, filter = nil, show all processes |
| Unknown field name | Parse error: "unknown field: X (valid: name, user, pid, cpu, mem, conn)" |
| Unterminated quote | Parse error: "unterminated quote" |
| Very long query | Text input handles scrolling; no length limit enforced |
| Special characters | Handled gracefully; quotes allow spaces in values |
| Case sensitivity | Fields and operators case-insensitive; AND/OR case-insensitive |
| Cursor during filter | Clamped to filtered result count on each keystroke |
| Toggle grouping while filtered | Works correctly; filter applies to grouped results |
| Filter locked + Esc | First Esc clears filter; queryMode already false |

---

## 16. File-by-File Reference

### 16.1 Files Created in This Implementation

| File | Lines | Purpose |
|------|-------|---------|
| `internal/metrics/connections.go` | 21 | PID → connection count map via single `net.Connections("all")` syscall |
| `internal/metrics/detail.go` | 133 | On-demand deep process inspection with per-field error accumulation |
| `internal/ui/panels/summary.go` | 76 | 3-line compact system summary (CPU bar + per-core, MEM bar, Swap bar) |
| `internal/ui/panels/proctable.go` | 146 | Scrollable, selectable, sortable process table with 7 columns |
| `internal/ui/panels/nettop.go` | 92 | Compact network rates + top 5 processes by connection count |
| `internal/ui/panels/procdetail.go` | 126 | Full detail view: info, sparklines, environment |
| `internal/ui/query.go` | ~360 | Query types, DSL parser, filter matching logic |
| `internal/ui/query_test.go` | ~220 | Comprehensive tests for query parser and matcher |
| `internal/ui/panels/querybar.go` | ~60 | Query/search input bar renderer |

### 16.2 Files Modified in This Implementation

| File | Change Summary |
|------|---------------|
| `internal/metrics/collector.go` | Added `MemRSS uint64` and `ConnCount int` to `ProcessInfo`; changed `collectProcesses(10)` → `collectProcesses(25)` |
| `internal/metrics/process.go` | Added `p.MemoryInfo()` for RSS; set `ConnCount: -1` sentinel |
| `internal/ui/model.go` | Added query state fields, `handleQueryInput()`, `applyQueryFilter()`, `filteredProcesses()`, `/` and `Q` key handlers |
| `internal/ui/layout.go` | Added `queryBar` parameter to `RenderDashboardLayout()` |
| `internal/ui/panels/help.go` | Added `/` and `Q` keybindings to compact and expanded help |
| `internal/ui/styles/styles.go` | Added `ColorConn`, `CursorRowStyle`, `SortIndicatorStyle`, `DetailHeaderStyle` |
| `go.mod` | Added `charmbracelet/bubbles v0.21.1` as direct dependency for textinput |

### 16.3 Files Retained (Legacy, Unreferenced from New View)

| File | Status |
|------|--------|
| `internal/ui/panels/cpu.go` | Kept for reference; not called from new `View()` |
| `internal/ui/panels/memory.go` | Kept for reference; not called from new `View()` |
| `internal/ui/panels/disk.go` | Kept for reference; not called from new `View()` |
| `internal/ui/panels/network.go` | Kept for reference; not called from new `View()` |
| `internal/ui/panels/process.go` | Kept for reference; not called from new `View()` |
