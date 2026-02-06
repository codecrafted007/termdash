# termdash

A modern, interactive terminal-based system monitor built with Go. Think `htop` meets powerful query capabilities.

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)
![Platform](https://img.shields.io/badge/Platform-macOS%20|%20Linux-lightgrey)

## Features

- **Real-time Monitoring** — CPU, memory, swap, disk, and network metrics updated every 2 seconds
- **Interactive Process Table** — Navigate, sort, and inspect processes with vim-style keybindings
- **Process Detail View** — Deep dive into any process with CPU/memory sparklines, environment variables, file descriptors, and more
- **Process Grouping** — Group processes by name to see aggregate resource usage
- **Network Connections** — Per-process connection counts at a glance
- **Powerful Query System** — Filter processes using simple search or a full query DSL
- **Export Capabilities** — Export snapshots to JSON or record continuous data to CSV

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
| `e` | Export snapshot to JSON |
| `E` | Toggle CSV recording |
| `?` | Toggle help |
| `q` | Quit |

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

## Building

```bash
# Build
make build

# Run
make run

# Test
make test

# Clean
make clean
```

## Project Structure

```
termdash/
├── cmd/termdash/          # Application entry point
├── internal/
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
│       ├── layout.go      # Layout composition
│       ├── panels/        # UI panel renderers
│       └── styles/        # Color palette and styles
└── docs/
    ├── HLD.md             # High-level design
    └── LLD.md             # Low-level design
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
