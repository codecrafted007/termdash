# termdash

A modern, interactive terminal-based system monitor built with Go. Think `htop` meets powerful query capabilities.

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)
![Platform](https://img.shields.io/badge/Platform-macOS%20|%20Linux-lightgrey)

![termdash demo](demo.gif)

## Why termdash?

### The Problem

You're debugging a production issue. CPU is spiking. You need to find all Java processes consuming more than 50% CPU that have active network connections. With traditional tools, you'd do something like:

```bash
ps aux | grep java | awk '$3 > 50' | while read line; do
  pid=$(echo $line | awk '{print $2}')
  netstat -tlnp 2>/dev/null | grep $pid
done
```

Clunky. Error-prone. And by the time you've typed it, the moment has passed.

### The Solution

With termdash, you press `Q` and type:

```
name ~ java and cpu > 50 and conn > 0
```

**That's it.** Real-time, filtered results. No pipes. No awk. No context switching.

### Motivation

Every developer and sysadmin knows the drill: you're SSHed into a server, something's wrong, and you need answers *fast*. Traditional tools like `htop` are great for general monitoring, but when you need to find specific processes matching complex criteria, you end up juggling `ps`, `grep`, `awk`, and `netstat` in increasingly creative combinations.

termdash was born from a simple idea: **what if your process monitor could speak your language?**

Instead of memorizing command-line incantations, you describe what you're looking for:
- "Show me all processes owned by root with more than 5 connections"
- "Find anything using more than 10% CPU and 5% memory"
- "Filter to just Python or Node.js processes"

## What Makes termdash Different

| Feature | htop | btop | gotop | **termdash** |
|---------|------|------|-------|--------------|
| Process filtering | Basic text search | Basic text search | None | **Full query DSL** |
| Query language | ❌ | ❌ | ❌ | ✅ `cpu > 50 and name ~ java` |
| Vim-style search | ❌ | ❌ | ❌ | ✅ Press `/` to search |
| Process grouping | ❌ | ❌ | ❌ | ✅ Aggregate by name |
| Per-process connections | ❌ | ❌ | ❌ | ✅ Built-in |
| Export to JSON/CSV | ❌ | ❌ | ❌ | ✅ One keypress |
| Historical replay | ❌ | ❌ | ❌ | ✅ SQLite-backed time travel (`t` to replay) |
| Process detail view | Limited | Limited | ❌ | ✅ Full inspection |
| Sparkline history | ❌ | ✅ | ✅ | ✅ Per-process |
| Written in | C | C++ | Go | **Go** |

### Key Differentiators

1. **Query DSL** — The killer feature. No other terminal monitor lets you write `user = root and conn > 0 and cpu > 10`. Filter processes like you query a database.

2. **Vim-Style Workflow** — Press `/` for quick search, `j/k` to navigate, `Enter` to inspect. Feels like home for terminal users.

3. **Process Grouping** — See 47 Chrome processes? Toggle grouping with `p` to see them as one entry with aggregate CPU/memory.

4. **Connection Awareness** — Every process shows its network connection count. Filter by it. Sort by it. Essential for debugging networked applications.

5. **Export Everything** — Press `e` for JSON snapshot, `E` to start recording CSV. Perfect for post-incident analysis or automation.

6. **Historical Replay** — Every terminal monitor is strictly live — once a spike passes, the evidence is gone. termdash is different. It records snapshots to a local SQLite database every 10 seconds. Press `t` to enter replay mode and scroll through past system state. An alert fired at 3 AM? SSH in the morning and replay exactly what happened. Intermittent CPU spikes? Scroll back and catch every one. ~25 MB/day with defaults, zero config required.

7. **Modern Codebase** — Built with Go and the Charm ecosystem. Easy to understand, extend, and contribute to.

## Features

- **Real-time Monitoring** — CPU, memory, swap, disk, and network metrics updated every 2 seconds
- **Interactive Process Table** — Navigate, sort, and inspect processes with vim-style keybindings
- **Process Detail View** — Deep dive into any process with CPU/memory sparklines, environment variables, file descriptors, and more
- **Process Grouping** — Group processes by name to see aggregate resource usage
- **Network Connections** — Per-process connection counts at a glance
- **Powerful Query System** — Filter processes using simple search or a full query DSL
- **Export Capabilities** — Export snapshots to JSON or record continuous data to CSV
- **Historical Replay** — SQLite-backed snapshot history with time-travel replay mode

## Installation

### From Source

```bash
git clone https://github.com/codecrafted007/termdash.git
cd termdash
go build -o bin/termdash ./cmd/termdash
./bin/termdash
```

### Using Go Install

```bash
go install github.com/codecrafted007/termdash/cmd/termdash@latest
```

## Usage

```bash
termdash
```

### Configuration

termdash supports a `.td` config file to customize the dashboard layout and refresh rate. By default it looks for `~/.config/termdash/config.td`.

```bash
# Use default config path
termdash

# Use a specific config file
termdash --config ./my-config.td
termdash -c ./my-config.td
```

#### Config Format

Config files use HCL syntax with `global` settings and a `layout` grid of rows and columns:

```hcl
global {
  refresh = "1s"
  title   = "dev"
}

layout {
  row {
    weight = 1
    col {
      weight = 1
      widget = "cpu"
    }
    col {
      weight = 1
      widget = "memory"
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
```

- **`refresh`** — How often metrics update (`"1s"`, `"2s"`, `"500ms"`, etc.)
- **`title`** — Dashboard title shown in the header
- **`weight`** — Proportional size of rows/columns (higher = larger)
- **`widget`** — Which panel to render: `"cpu"`, `"memory"`, `"summary"`, `"procs"`, `"nettop"`

#### History and Replay Settings

termdash records system snapshots to a local SQLite database, enabling you to replay past system state. This is configured in the `global {}` block:

```hcl
global {
  refresh          = "2s"
  title            = "dev"
  history          = "24h"    # how long to keep data. "0" disables history entirely.
  history_interval = "10s"    # how often to write a snapshot to the DB
  history_procs    = 50       # max processes per snapshot (by CPU). 0 = all.
  db_path          = ""       # custom DB path. default: ~/.local/share/termdash/history.db
}
```

| Setting | Default | Description |
|---------|---------|-------------|
| `history` | `"24h"` | Retention period. Accepts any Go duration (`"24h"`, `"168h"` for 7 days). Set to `"0"` to disable history completely — no DB file will be created. |
| `history_interval` | `"10s"` | Write frequency. Lower values give finer replay resolution but use more disk. Minimum `"2s"`. |
| `history_procs` | `50` | Max processes stored per snapshot, sorted by CPU usage. Set to `0` to store all processes. |
| `db_path` | `""` | Path to the SQLite database file. When empty (default), uses `~/.local/share/termdash/history.db`. The directory is created automatically. |

**Storage usage:** With defaults (10s interval, 50 processes, 24h retention), the database uses approximately **25 MB per day**.

#### Available Widgets

| Widget | Description |
|--------|-------------|
| `cpu` | CPU usage with per-core breakdown and sparkline |
| `memory` | Memory and swap usage bars |
| `summary` | Combined CPU, memory, and swap overview |
| `procs` | Interactive process table (sort, search, inspect) |
| `nettop` | Network I/O rates and top talkers |

#### Preset Configs

Ready-to-use configs are included in `configs/`. Copy one or use it directly:

```bash
# Developer — CPU + Memory on top, large process table. 1s refresh.
termdash -c configs/dev.td

# Ops / SysAdmin — Balanced 3-panel top row, procs + nettop below. 2s refresh.
termdash -c configs/ops.td

# Minimal — Just the process table, nothing else. 3s refresh.
termdash -c configs/minimal.td

# Network — Network top panel + summary on top, processes below. 1s refresh.
termdash -c configs/network.td
```

To make one your default:

```bash
mkdir -p ~/.config/termdash
cp configs/dev.td ~/.config/termdash/config.td
```

If no config file exists, termdash uses a built-in default layout (summary + nettop on top, procs on bottom, 2s refresh).

### Keyboard Shortcuts

#### Dashboard View

| Key | Action |
|-----|--------|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `g` / `Home` | Jump to top |
| `G` / `End` | Jump to bottom |
| `Enter` | Open process detail view |
| `s` | Cycle sort column (CPU% → MEM% → PID → NAME → CONN) |
| `S` | Toggle sort direction |
| `p` | Toggle process grouping |
| `/` | Search by name or PID |
| `Q` | Open query filter (DSL) |
| `t` | Enter replay mode |
| `e` | Export snapshot to JSON |
| `E` | Toggle CSV recording |
| `?` | Toggle help |
| `q` | Quit |

#### Replay Mode

Press `t` in the dashboard to enter replay mode. The header changes to show a timeline position and the snapshot timestamp.

| Key | Action |
|-----|--------|
| `[` / `Left` | Step back one snapshot (~10s) |
| `]` / `Right` | Step forward one snapshot |
| `{` | Jump back ~1 minute |
| `}` | Jump forward ~1 minute |
| `j` / `k` | Navigate process table within the snapshot |
| `Esc` / `t` | Exit replay, return to live view |

#### Process Detail View

| Key | Action |
|-----|--------|
| `Esc` | Return to dashboard |
| `e` | Export snapshot to JSON |
| `E` | Toggle CSV recording |
| `?` | Toggle help |
| `q` | Quit |

#### Query/Search Mode

| Key | Action |
|-----|--------|
| `Enter` | Lock filter and return to dashboard |
| `Esc` | Clear filter and exit |

## Query DSL

termdash includes a powerful query language for filtering processes.

### Simple Search (`/`)

Press `/` and type to search:
```
chrome          → processes with "chrome" in name
1234            → process with PID 1234 or "1234" in name
```

### Field Query (`Q`)

Press `Q` to enter query mode with full DSL support:

#### Supported Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Process name |
| `user` | string | Username |
| `pid` | int | Process ID |
| `cpu` | float | CPU percentage |
| `mem` | float | Memory percentage |
| `conn` | int | Connection count |

#### Operators

| Operator | Meaning | Example |
|----------|---------|---------|
| `=` | Equals | `user = root` |
| `!=` | Not equals | `name != chrome` |
| `>` | Greater than | `cpu > 50` |
| `<` | Less than | `mem < 10` |
| `>=` | Greater or equal | `conn >= 5` |
| `<=` | Less or equal | `cpu <= 1` |
| `~` | Contains | `name ~ java` |

#### Logical Operators

Combine conditions with `and` / `or`:

```
name ~ nginx                      # Processes containing "nginx"
cpu > 50                          # High CPU processes
user = root and conn > 0          # Root processes with connections
name ~ java or name ~ python      # Java or Python processes
mem > 5 and cpu > 10              # Memory and CPU intensive processes
```

## Screenshots

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ ⣿ termdash  Host: macbook  OS: darwin 15.0  Uptime: 5d 3h 12m  14:32:05    │
├─────────────────────────────────────────────────────────────────────────────┤
│ CPU [████████░░░░░░░░░░░░] 32%  0:45%  1:28%  2:35%  3:22%                  │
│ Mem [██████████████░░░░░░]  12.4/16.0 GiB (78%)                             │
│ Swp [░░░░░░░░░░░░░░░░░░░░]  0.0/2.0 GiB (0%)                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ PID▼    NAME              USER        CPU%   MEM%   RSS      CONN          │
│ ──────────────────────────────────────────────────────────────────          │
│ > 1234  chrome            brajesh     12.4    5.2   1.2G      42           │
│   5678  code              brajesh      8.1    3.8   890M      15           │
│   9012  node              brajesh      4.2    2.1   512M      23           │
│   ...                                                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ Net: ▲1.2 MiB/s ▼3.4 MiB/s │ Top: chrome(42) node(23) code(15)             │
│ j/k:move  /:search  Q:query  s:sort  p:group  e:export  ?:help  q:quit     │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Tech Stack

- **[Bubbletea](https://github.com/charmbracelet/bubbletea)** — Terminal UI framework (Elm architecture)
- **[Bubbles](https://github.com/charmbracelet/bubbles)** — TUI components
- **[Lipgloss](https://github.com/charmbracelet/lipgloss)** — Style definitions
- **[ntcharts](https://github.com/NimbleMarkets/ntcharts)** — Terminal sparkline charts
- **[gopsutil](https://github.com/shirou/gopsutil)** — Cross-platform system metrics
- **[modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)** — Pure-Go SQLite for history persistence (no CGO required)

## Building

### Quick Build

```bash
# Build for current platform
make build

# Build and run
make run

# Run tests
make test

# Clean build artifacts
make clean
```

### Cross-Platform Builds

Build for all supported platforms:

```bash
# Build for all platforms (Linux, macOS, Windows)
make build-all

# Or build for specific platforms
make build-linux          # Linux amd64
make build-linux-arm64    # Linux arm64 (Raspberry Pi, etc.)
make build-darwin         # macOS Intel
make build-darwin-arm64   # macOS Apple Silicon
make build-windows        # Windows amd64
```

### Docker Builds

Build without installing Go locally:

```bash
# Build using Docker (outputs Linux amd64 binary)
make docker-build

# Build Docker image
make docker-image

# Run in Docker container
make docker-run

# Build for specific platform using Docker
./scripts/docker-build.sh linux arm64 v1.0.0
./scripts/docker-build.sh darwin amd64 v1.0.0
```

### Release

Create release artifacts for all platforms:

```bash
# Build all platforms and create .tar.gz/.zip archives
make release VERSION=v1.0.0
```

Output binaries are placed in the `dist/` directory.

## Project Structure

```
termdash/
├── cmd/termdash/          # Application entry point
├── configs/               # Preset .td config files
│   ├── dev.td             # Developer layout
│   ├── ops.td             # Ops/SysAdmin layout
│   ├── minimal.td         # Minimal (procs only)
│   └── network.td         # Network-focused layout
├── pkg/dsl/               # Config DSL parser (HCL-based)
│   ├── config.go          # Config structs with HCL tags
│   ├── parser.go          # HCL file parsing + LoadConfig
│   ├── defaults.go        # Default config (fallback)
│   └── validate.go        # Config validation
├── internal/
│   ├── history/           # SQLite-backed snapshot persistence
│   │   ├── store.go       # DB open/close, write/read/cleanup
│   │   ├── writer.go      # Bubble Tea Cmd helpers
│   │   └── store_test.go  # Store tests
│   ├── metrics/           # System metrics collection
│   │   ├── collector.go   # Main collector orchestration
│   │   ├── cpu.go         # CPU metrics
│   │   ├── memory.go      # Memory/swap metrics
│   │   ├── disk.go        # Disk usage
│   │   ├── network.go     # Network I/O
│   │   ├── process.go     # Process listing
│   │   ├── connections.go # Network connections per process
│   │   └── detail.go      # Detailed process info
│   └── ui/
│       ├── model.go       # Bubbletea model (state machine)
│       ├── query.go       # Query parser and filter
│       ├── layout.go      # Layout composition (config-driven)
│       ├── panels/        # UI panel renderers
│       └── styles/        # Color palette and styles
└── docs/
    ├── HLD.md             # High-level design
    ├── LLD.md             # Low-level design
    └── DSL_SPEC.md        # DSL config language spec
```

## Requirements

- Go 1.25 or later
- macOS or Linux (Windows support via WSL)
- Terminal with 256-color support recommended

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [htop](https://htop.dev/), [btop](https://github.com/aristocratos/btop), and [gotop](https://github.com/xxxserxxx/gotop)
- Built with the excellent [Charm](https://charm.sh/) ecosystem

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/codecrafted007">Brajesh</a>
</p>
