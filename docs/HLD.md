# High-Level Design: termdash Interactive Process Monitor

## 1. Overview

termdash is an interactive, htop-style terminal process monitor built in Go. It provides real-time system metrics (CPU, memory, swap, disk, network) alongside a scrollable, sortable process table with per-process network connection counts. Users can drill into any process for a full detail view with CPU/memory sparkline history, environment variables, file descriptors, and thread counts.

The application runs inside an alternate-screen terminal session and refreshes every 2 seconds using an event-driven architecture built on the Bubbletea TUI framework.

---

## 2. Goals and Non-Goals

### Goals
- Real-time system monitoring with sub-3-second latency
- Interactive process table with cursor navigation, multi-column sorting, and scrolling
- Per-process network connection counting via a single `net.Connections("all")` syscall
- On-demand process detail view with CPU/memory history sparklines, environment, FDs, threads
- Interactive query/search capability for filtering processes by name, user, PID, CPU, memory, or connections
- Graceful degradation when running without root (connections and FDs show "N/A" / "-")
- Responsive terminal layout that adapts to window resizes
- Zero additional dependencies beyond what the project already uses (gopsutil, ntcharts, bubbletea, lipgloss, bubbles)

### Non-Goals
- Persistent logging or metric export
- Remote monitoring or networked operation
- Process management (kill, renice, signal sending)
- Disk I/O per-process attribution
- Configuration file or CLI flags

---

## 3. Architecture Overview

```
┌──────────────────────────────────────────────────────┐
│                     Terminal (TTY)                    │
│  ┌────────────────────────────────────────────────┐  │
│  │              Bubbletea Runtime                  │  │
│  │  ┌──────────┐  ┌───────────┐  ┌────────────┐  │  │
│  │  │ Input    │  │  Model    │  │  Renderer  │  │  │
│  │  │ (Keys,   │→ │ (State +  │→ │  (View()   │  │  │
│  │  │  Resize) │  │  Update) │  │   → string)│  │  │
│  │  └──────────┘  └─────┬─────┘  └────────────┘  │  │
│  │                      │                         │  │
│  │          ┌───────────┴───────────┐             │  │
│  │          │    tea.Cmd Layer      │             │  │
│  │          │  (Async Side Effects) │             │  │
│  │          └───────────┬───────────┘             │  │
│  └──────────────────────┼─────────────────────────┘  │
│                         │                            │
│  ┌──────────────────────┴─────────────────────────┐  │
│  │              Metrics Layer                      │  │
│  │  ┌──────────┐ ┌────────────┐ ┌──────────────┐  │  │
│  │  │Collector │ │Connections │ │ProcessDetail │  │  │
│  │  │(Snapshot)│ │ (ConnMap)  │ │ (On-Demand)  │  │  │
│  │  └────┬─────┘ └─────┬──────┘ └──────┬───────┘  │  │
│  │       │              │               │          │  │
│  │  ┌────┴──────────────┴───────────────┴───────┐  │  │
│  │  │           gopsutil v3 (OS Abstraction)    │  │  │
│  │  │   cpu · mem · disk · net · process · host │  │  │
│  │  └───────────────────────────────────────────┘  │  │
│  └─────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

### Layer Summary

| Layer | Responsibility | Key Packages |
|-------|---------------|--------------|
| **Terminal** | Raw I/O, alternate screen, key events | bubbletea runtime |
| **UI Model** | State machine, message routing, view dispatch | `internal/ui` |
| **Panel Renderers** | Stateless string rendering functions | `internal/ui/panels` |
| **Styles** | Centralized color palette and lipgloss styles | `internal/ui/styles` |
| **Metrics** | System data collection, rate calculation | `internal/metrics` |
| **OS Abstraction** | Cross-platform syscall wrappers | gopsutil v3 |

---

## 4. Component Diagram

```
┌─────────────┐     ┌──────────────────────────────────────────────┐
│  main.go    │────→│  Model (internal/ui/model.go)                │
│  (entry)    │     │                                              │
└─────────────┘     │  State:                                      │
                    │  ├─ viewState: Dashboard | ProcessDetail     │
                    │  ├─ snapshot: Snapshot (system metrics)      │
                    │  ├─ cursor, scrollOffset, sortColumn         │
                    │  ├─ connCounts: map[PID]→count               │
                    │  ├─ cpuHistory, sendHistory, recvHistory     │
                    │  └─ detailPID, detailInfo, detailHistories   │
                    │                                              │
                    │  Messages In:                                │
                    │  ├─ tickMsg (2s interval)                    │
                    │  ├─ snapshotMsg (full system metrics)        │
                    │  ├─ connCountMsg (PID→conn count map)        │
                    │  ├─ processDetailMsg (single process detail) │
                    │  ├─ detailTickMsg (1s interval, detail only) │
                    │  ├─ tea.KeyMsg (keyboard input)              │
                    │  └─ tea.WindowSizeMsg (terminal resize)      │
                    │                                              │
                    │  Commands Out:                               │
                    │  ├─ collectCmd → snapshotMsg                 │
                    │  ├─ collectConnCountsCmd → connCountMsg      │
                    │  ├─ collectDetailCmd → processDetailMsg      │
                    │  ├─ tickCmd → tickMsg                        │
                    │  └─ detailTickCmd → detailTickMsg            │
                    └──────────────┬───────────────────────────────┘
                                   │
                    ┌──────────────┴───────────────┐
              ┌─────┤         View()               ├─────┐
              │     └──────────────────────────────┘     │
              ▼                                          ▼
┌─────────────────────────┐              ┌─────────────────────────┐
│   Dashboard View        │              │   Detail View           │
│   ├─ RenderHeader       │              │   ├─ RenderProcessDet.  │
│   ├─ RenderSummary      │              │   └─ RenderHelp(detail) │
│   ├─ RenderProcessTable │              └─────────────────────────┘
│   ├─ RenderNetTop       │
│   └─ RenderHelp(dash)   │
└─────────────────────────┘
```

---

## 5. Data Flow

### 5.1 Main Collection Loop

```
Init() ──→ tickCmd() + collectCmd()
             │              │
             ▼              ▼
        tickMsg(2s)    snapshotMsg(data)
             │              │
             │              ├─→ Update histories (cpu, net send/recv)
             │              ├─→ Clamp cursor to process list bounds
             │              └─→ Dispatch: tickCmd() + collectConnCountsCmd()
             │                                              │
             └──────→ collectCmd() ◄────────────────────    ▼
                                                      connCountMsg(map)
                                                            │
                                                            └─→ Store in model
```

### 5.2 Detail View Lifecycle

```
User presses Enter on process row
    │
    ├─→ Set viewState = ViewProcessDetail
    ├─→ Create fresh CPU + MEM history buffers
    ├─→ Dispatch: collectDetailCmd(pid) + detailTickCmd(1s)
    │
    ▼
processDetailMsg arrives
    │
    ├─→ Store detail info
    ├─→ Push CPU% and MEM% into detail history buffers
    │
    ▼
detailTickMsg(1s) ──→ collectDetailCmd(pid) + detailTickCmd(1s)
    │                       │
    └───────────────────────┘  (repeats while in detail view)

User presses Esc
    │
    ├─→ Set viewState = ViewDashboard
    ├─→ Nil out detail info + history buffers
    └─→ Detail tick stops (guard in Update: viewState != ViewProcessDetail → noop)
```

### 5.3 Connection Collection (Separate Path)

```
snapshotMsg handler
    │
    └─→ collectConnCountsCmd() (parallel with tickCmd)
            │
            └─→ psnet.Connections("all") — single syscall
                    │
                    ├─→ Group by PID → map[int32]int
                    └─→ Return as connCountMsg
                            │
                            └─→ Stored in model.connCounts
                                Applied to processes at sort/render time
```

### 5.4 Query/Search Flow

```
User presses '/' or 'Q'
    │
    ├─→ queryMode = true
    ├─→ queryType = QuerySearch (/) or QueryFilter (Q)
    ├─→ Focus text input, show query bar
    │
    ▼
User types query
    │
    ├─→ Update queryInput value
    ├─→ Parse query (DSL mode) or use as-is (search mode)
    │   ├─→ ParseQuery() → FilterNode AST (for DSL)
    │   └─→ On error: show in queryError, filter = nil
    ├─→ Apply filter to process list in real-time
    └─→ Clamp cursor to filtered results

User presses Enter
    │
    ├─→ filterLocked = true
    ├─→ queryMode = false
    └─→ Filter remains active, shown in status

User presses Esc
    │
    ├─→ If filterLocked: clear filter, unlock
    ├─→ queryMode = false
    └─→ Return to unfiltered view
```

---

## 6. View Architecture

### 6.1 Dashboard View (Main View)

```
┌───────────────────────────────────── width ──────────────────────────────────┐
│ Header: hostname, OS, uptime, time                                1 line    │
│ CPU [████░░░░░░] 32%  0:45%  1:28%  2:35%                        1 line    │
│ Mem [██████░░░░]  6.2/16.0 GiB (54%)                             1 line    │
│ Swp [░░░░░░░░░░]  0.0/2.0 GiB (0%)                              1 line    │
│ PID▼   NAME          USER    CPU%  MEM%  RSS      CONN           2 lines   │
│ ─────────────────────────────────────────────────────             (header+  │
│ > 123 chrome        brajesh  4.2   3.1  512M   15    ◄ cursor     sep)     │
│   456 code          brajesh  3.8   2.8  448M    8                          │
│   789 node          brajesh  2.1   1.5  256M   12   visibleRows  │
│   ...                                            (dynamic)       │
│ Query: cpu > 10 and name ~ chrome                      1 line    │ ◄ optional
│ Net: ▲1.2 MiB/s ▼3.4 MiB/s │ Top: chrome(15) node(12)  1 line  │
│ j/k:move  /:search  Q:query  s:sort  ?:help  q:quit     1 line  │
└──────────────────────────────────────────────────────────────────┘
```

**Dynamic row calculation:** `visibleRows = termHeight - 11` (header + summary 3 + table header 2 + netTop 1 + help 1 + padding 2). NetTop collapses when `visibleRows < 5`, reclaiming 2 rows. Query bar appears between process table and netTop when query mode is active or filter is locked.

### 6.2 Detail View

```
┌───────────────────────────────────── width ──────────────────────────────────┐
│ Process: chrome (PID 12345)  ── [Esc] back              title bar          │
│ User: brajesh       Status: R     Threads: 42   FDs: 128                   │
│ Parent: 1           Nice: 0       Conns: 15     Created: 2025-01-15 10:23  │
│ Cmd: /usr/bin/google-chrome --flag1 --flag2                                │
│ ────────────────────────────────────────────────────────────                │
│ CPU History          │ Memory History                                       │
│ ▁▂▃▅▇█▇▅▃▂▁▂▃▅▇     │ ▁▁▂▂▃▃▄▅▆▇████▇                                    │
│ Current: 45.2%       │ RSS: 1.2 GiB  VMS: 3.4 GiB                         │
│ ────────────────────────────────────────────────────────────                │
│ Environment:                                                               │
│   HOME=/Users/brajesh                                                      │
│   PATH=/usr/local/bin:/usr/bin:...                                         │
│   ... and 38 more                                                          │
│ Esc:back  ?:help  q:quit                                                   │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Key Design Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | **Connections collected separately** from main snapshot | `net.Connections("all")` can be slow and may fail without root. Separate command prevents blocking the main collection pipeline. |
| 2 | **Detail collected on-demand only** | Environment, cmdline, FDs, and thread data are only fetched when entering detail view — not every 2s tick. Avoids unnecessary syscalls for 25 processes. |
| 3 | **Sort at render time** | Collector always returns CPU%-sorted top 25. User's sort preference is applied in `sortedProcesses()` within `View()` before passing to the renderer. This keeps the collector simple and the sort order a pure UI concern. |
| 4 | **History buffers for selected process only** | CPU and memory history buffers are created on detail entry and destroyed on exit. No memory overhead for unviewed processes. Main dashboard retains only 3 global histories (total CPU, net send, net recv). |
| 5 | **No new dependencies** | gopsutil already provides `net.Connections`, `process.Environ`, `process.NumFDs`, etc. ntcharts provides sparklines. bubbles provides text input. No new modules added to `go.mod`. |
| 6 | **Stateless panel renderers** | Every `Render*()` function is a pure function: data in, styled string out. No internal state, no side effects. Testable in isolation. |
| 7 | **Value-receiver Model** | Follows the Bubbletea convention: `Update()` returns a new `Model` value. Pointer receivers used only for `ensureCursorVisible()` which mutates the local copy before return. |
| 8 | **Duplicate SortColumn enum in panels** | Avoids an import cycle between `ui` and `panels`. The panels package defines its own `SortColumn` with identical iota values, cast via `panels.SortColumn(m.sortColumn)`. |
| 9 | **Filter applied at render time** | Query filtering follows the same pattern as sorting — `filteredProcesses()` wraps `sortedProcesses()` and is called in `View()`. Real-time filtering as user types without explicit "apply" action. |
| 10 | **Recursive descent parser for query DSL** | Simple, readable implementation for the query language. Handles AND/OR precedence (AND binds tighter than OR). No external parser dependency needed. |

---

## 8. Technology Stack

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| Language | Go | 1.25.6 | Application runtime |
| TUI Framework | charmbracelet/bubbletea | v1.3.10 | Elm-architecture terminal UI |
| UI Components | charmbracelet/bubbles | v0.21.1 | Text input for query bar |
| Styling | charmbracelet/lipgloss | v1.1.0 | ANSI color and layout composition |
| Charts | NimbleMarkets/ntcharts | v0.4.0 | Sparkline rendering |
| System Metrics | shirou/gopsutil/v3 | v3.24.5 | Cross-platform OS metric collection |

---

## 9. Package Structure

```
termdash/
├── cmd/termdash/
│   └── main.go                    # Entry point: creates Model, runs Bubbletea
│
├── internal/
│   ├── metrics/                   # Data collection layer (no UI dependency)
│   │   ├── collector.go           # Types (Snapshot, ProcessInfo, etc.) + orchestration
│   │   ├── cpu.go                 # Per-core and total CPU via gopsutil
│   │   ├── memory.go              # RAM + Swap via gopsutil
│   │   ├── disk.go                # Per-mount disk usage with FS filtering
│   │   ├── network.go             # Network I/O rates + host info
│   │   ├── process.go             # Top 25 processes by CPU (with RSS)
│   │   ├── connections.go         # PID → connection count map
│   │   ├── detail.go              # On-demand single-process deep inspection
│   │   ├── format.go              # FormatBytes, FormatRate utilities
│   │   └── *_test.go              # Unit tests
│   │
│   └── ui/                        # Presentation layer
│       ├── model.go               # Bubbletea Model: state, Update, View, commands
│       ├── layout.go              # Layout composition (dashboard + detail)
│       ├── query.go               # Query parser, filter types, matching logic
│       ├── grouping.go            # Process grouping logic
│       ├── model_test.go          # Model unit tests
│       ├── query_test.go          # Query parser tests
│       ├── panels/                # Stateless panel renderer functions
│       │   ├── header.go          # System header bar
│       │   ├── summary.go         # Compact CPU/MEM/Swap bars
│       │   ├── proctable.go       # Scrollable, sortable process table
│       │   ├── nettop.go          # Network rates + top consumers
│       │   ├── procdetail.go      # Full process detail view
│       │   ├── querybar.go        # Query/search input bar
│       │   ├── help.go            # Context-aware keybinding footer
│       │   ├── cpu.go             # (Legacy) Full CPU panel with sparkline
│       │   ├── memory.go          # (Legacy) Full memory panel
│       │   ├── disk.go            # (Legacy) Disk usage panel
│       │   ├── network.go         # (Legacy) Network panel with sparklines
│       │   └── process.go         # (Legacy) Simple process table
│       └── styles/
│           └── styles.go          # Color palette + reusable styles
│
├── go.mod
├── go.sum
└── Makefile
```

---

## 10. Error Handling Strategy

| Scenario | Handling |
|----------|----------|
| `Collect()` fails | `errMsg` sent to model, displayed to user, tick continues |
| `net.Connections("all")` fails (no root) | Returns empty map; CONN column shows "-" |
| `process.NumFDs()` fails (macOS sandboxing) | `NumFDs = -1`; detail view shows "N/A" |
| `process.Environ()` fails (permissions) | Empty list; detail shows "(not available)" |
| Process exits during detail view | `collectDetailCmd` returns error → auto-return to dashboard |
| Individual field in `CollectProcessDetail` fails | Error accumulated in `Err` string; other fields still populated |
| Terminal too small | `visibleRows` clamped to minimum 1; netTop collapses if < 5 rows |

---

## 11. Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| Main refresh interval | 2 seconds | Balances freshness vs. CPU overhead |
| Detail refresh interval | 1 second | Finer granularity for sparklines |
| Process list size | Top 25 by CPU% | Collector filters; sort is render-time |
| History buffer capacity | 60 samples | ~2 min of data at 2s intervals |
| Connection collection | Single syscall | `net.Connections("all")` returns all, grouped by PID in O(n) |
| Detail collection | On-demand only | ~15 gopsutil calls per process, only when viewing |
| Memory overhead (idle) | 3 History buffers x 60 floats | ~1.4 KiB for histories |
| Memory overhead (detail) | +2 History buffers x 60 floats | ~960 bytes, freed on exit |

---

## 12. Security Considerations

- **No privilege escalation**: termdash reads `/proc` (Linux) or uses macOS sysctl. It never requires or requests root.
- **Environment variable display**: Detail view shows process environment variables, which may contain secrets. This is equivalent to `ps eww` or `/proc/PID/environ` — read-access is governed by OS permissions.
- **No network listeners**: termdash is a purely local, read-only monitoring tool. It opens no sockets.
- **Truncation guards**: Environment variables are truncated to 200 chars and capped at 50 entries. Cmdline is truncated to terminal width. This prevents memory exhaustion from adversarial process names.

---

## 13. Historical Replay and SQLite Persistence

### 13.1 Motivation

Every terminal monitor — htop, btop, gotop — is strictly live. The moment a CPU spike, memory leak, or runaway process passes, the evidence is gone. You're left saying "something happened 10 minutes ago" with nothing to show for it.

This is a real problem:

- **Post-incident analysis:** An alert fires at 3 AM. By the time you SSH in, the spike has subsided. What process caused it? What was memory doing? Traditional monitors can't answer that.
- **Intermittent issues:** A process spikes CPU for 5 seconds every few minutes. You can't stare at the screen all day waiting to catch it. You need a record.
- **Correlation:** Was the network spike related to the CPU spike? You need to see both at the same point in time, after the fact.

No existing terminal monitor solves this. That's the gap.

termdash records system snapshots to a local SQLite database at a configurable interval (default: every 10s). Users can press `t` to enter replay mode and scroll through past system state — same dashboard, same process table, same data — just from a different point in time. The cost is minimal (~25 MB/day with defaults), and it can be disabled entirely with `history = "0"`.

### 13.2 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Bubbletea Model                          │
│                                                              │
│  WriteTickMsg (every history_interval)                       │
│       │                                                      │
│       ▼                                                      │
│  WriteCmd(store, snapshot, connCounts, maxProcs)             │
│       │                                                      │
│       ▼                                                      │
│  ┌──────────────────────────────────────────────────┐       │
│  │              history.Store                        │       │
│  │  ┌────────────┐  ┌────────┐  ┌──────────┐       │       │
│  │  │ snapshots  │  │ disks  │  │processes │       │       │
│  │  │ (system)   │  │ (per   │  │(top N by │       │       │
│  │  │            │  │  mount)│  │  CPU%)   │       │       │
│  │  └────────────┘  └────────┘  └──────────┘       │       │
│  │       SQLite (WAL mode, pure Go via modernc.org) │       │
│  └──────────────────────────────────────────────────┘       │
│                                                              │
│  CleanupCmd (on startup, deletes rows older than retention)  │
│                                                              │
│  Replay Mode:                                                │
│  LoadTimestampsCmd → TimestampsLoadMsg                       │
│  LoadSnapshotCmd   → ReplayLoadMsg                           │
│       │                                                      │
│       ▼                                                      │
│  replaySnapshot replaces m.snapshot in View()                │
└─────────────────────────────────────────────────────────────┘
```

### 13.3 Storage Schema

Three tables with CASCADE deletes from the parent `snapshots` table:

| Table | Content | Rows per snapshot |
|-------|---------|-------------------|
| `snapshots` | CPU, memory, swap, network, hostname, OS, uptime | 1 |
| `disks` | Per-mount-point usage | ~3 |
| `processes` | Top N processes by CPU (PID, name, user, CPU%, MEM%, RSS, conn count) | Up to `history_procs` (default 50) |

Pragmas: `journal_mode=WAL`, `synchronous=NORMAL`, `foreign_keys=ON`.

### 13.4 Storage Budget

Default settings: write every 10s, store top 50 processes, retain 24h.

| Component | Per snapshot | 24h (8,640 snapshots) |
|-----------|-------------|----------------------|
| System metrics (1 row) | ~200 bytes | 1.7 MB |
| Disk metrics (~3 rows) | ~180 bytes | 1.5 MB |
| Processes (50 rows) | ~2,500 bytes | 21.6 MB |
| **Total** | **~2.9 KB** | **~25 MB** |

### 13.5 Database Location

| Scenario | Path |
|----------|------|
| **Default** | `~/.local/share/termdash/history.db` |
| **Custom** | Set `db_path` in the `global {}` block of your `.td` config |
| **Disabled** | Set `history = "0"` — no database file is created |

The directory is created automatically on first run if it does not exist.

To override the default location:

```hcl
global {
  db_path = "/tmp/termdash-history.db"
}
```

### 13.6 Configuration

New fields in the `global {}` block:

| Field | Default | Description |
|-------|---------|-------------|
| `history` | `"24h"` | Retention period. Go duration (`"24h"`, `"7d"`, `"168h"`). Set to `"0"` to disable. |
| `history_interval` | `"10s"` | How often a snapshot is written to the DB. Minimum `"2s"`. |
| `history_procs` | `50` | Max processes stored per snapshot (sorted by CPU). `0` = all. |
| `db_path` | `""` | Custom DB file path. Empty = default (`~/.local/share/termdash/history.db`). |

### 13.7 Replay Mode

Replay mode lets users scroll through historical snapshots using the same dashboard interface.

**Entering:** Press `t` in the dashboard view. termdash loads all snapshot timestamps within the retention window and jumps to the most recent one.

**Navigation:**

| Key | Action |
|-----|--------|
| `[` / `Left` | Step back one snapshot (~10s) |
| `]` / `Right` | Step forward one snapshot |
| `{` | Jump back ~1 minute |
| `}` | Jump forward ~1 minute |
| `j` / `k` | Navigate the process table within the snapshot |
| `Esc` / `t` | Exit replay, return to live view |

**Header:** The normal header is replaced with a replay indicator showing the time offset, a timeline position bar, and the snapshot timestamp:

```
  REPLAY  -3m 20s  [-------#------------]  14:29:40   Press Esc to exit
```

**Data flow during replay:** `activeSnapshot()` and `activeConnCounts()` return replay data instead of live data. All rendering (summary, process table, nettop, widgets) uses these accessors, so the entire dashboard reflects the historical state.

### 13.8 Elm Architecture Compliance

All history I/O is performed through Bubble Tea `Cmd` functions — no goroutines are spawned directly:

| Cmd | Msg returned | Trigger |
|-----|-------------|---------|
| `WriteTick` | `WriteTickMsg` | Fires at `history_interval` |
| `WriteCmd` | `WriteResultMsg` | On each write tick (if live mode + ready) |
| `CleanupCmd` | `CleanupResultMsg` | Once on startup |
| `LoadTimestampsCmd` | `TimestampsLoadMsg` | On entering replay mode |
| `LoadSnapshotCmd` | `ReplayLoadMsg` | On each replay navigation step |

### 13.9 Error Handling

| Scenario | Behavior |
|----------|----------|
| DB file cannot be opened (permissions, disk full) | Warning printed to stderr; app runs without history |
| Write fails mid-session | `WriteResultMsg.Err` is non-nil; silently ignored, next tick retries |
| No history data when pressing `t` | Status message "No history available" shown for 3s |
| DB path directory doesn't exist | Created automatically by `os.MkdirAll` |

### 13.10 Dependency

`modernc.org/sqlite` — a pure-Go SQLite implementation. No CGO or C compiler required beyond what the project already needs for gopsutil on macOS.
