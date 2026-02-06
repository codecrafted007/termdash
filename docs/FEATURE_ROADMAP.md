# termdash Feature Roadmap: Differentiators vs htop/top

## Guiding Principle

Every feature here answers: "Why would someone choose termdash over htop?"
htop and top are mature, fast, and ubiquitous. termdash wins by combining tools
(htop + nethogs + iotop + lsof + docker stats) into a single binary with
capabilities none of them offer individually (historical replay, anomaly
detection, process grouping, export).

---

## Tier 1: Realistic and High-Impact

### 1. Per-Process Network Bandwidth (not just connection count)

**What htop lacks:** htop shows nothing about network. termdash already has
connection counts, but the real killer feature is actual bytes/sec per process.

**Implementation approach:**
- Linux: `/proc/PID/net/dev` + socket inode matching via `/proc/net/tcp{,6}` and
  `/proc/PID/fd/` symlink resolution. Map socket inodes to PIDs, then track
  bytes-transferred deltas per socket.
- macOS: `nettop`-style data via `NetworkStatistics` framework or periodic
  `lsof -i -n -P` parsing.
- Fallback: connection count (already implemented) when bandwidth isn't available.

**Display:**
- New columns in process table: `NET▲` (send rate), `NET▼` (recv rate)
- Detail view: per-process network sparkline history
- NetTop panel: replace aggregate-only with per-process bandwidth ranking

**Replaces:** `nethogs`

**Effort:** High — socket inode resolution is OS-specific and non-trivial.

---

### 2. Container / cgroup Awareness

**What htop lacks:** htop shows flat PIDs. Users running Docker, Podman, or
Kubernetes see raw process names with no container context.

**Implementation approach:**
- Linux: read `/proc/PID/cgroup` to extract container ID. Cross-reference with
  Docker API (`/var/run/docker.sock`) or parse
  `/sys/fs/cgroup/.../tasks` to map PIDs → container names.
- Detect container runtime: Docker, Podman, containerd, CRI-O.
- macOS: Docker Desktop runs in a VM, so container processes aren't visible
  as host PIDs. Show "N/A" or detect Docker Desktop via socket.

**Display:**
- New `CONTAINER` column in process table (truncated to 12 chars)
- Filter mode: press `c` to filter by container name
- Summary line: "Containers: 8 running, 2 paused"
- Detail view: show full container ID, image name, container status

**Replaces:** `docker stats`, `kubectl top pod`

**Effort:** Low-Medium — `/proc/PID/cgroup` parsing is straightforward on Linux.
Docker socket API adds moderate complexity.

---

### 3. Open File & Socket Inspector (built-in lsof)

**What htop lacks:** htop shows FD count but not what those FDs point to.
Users switch to `lsof -p PID` in another terminal.

**Implementation approach:**
- Linux: read `/proc/PID/fd/` directory, `readlink` each entry to get target
  (file path, `socket:[inode]`, `pipe:[inode]`, `anon_inode:...`).
  Resolve socket inodes via `/proc/net/tcp{,6}` and `/proc/net/unix`.
- macOS: use `proc_pidfdinfo` via cgo or shell out to `lsof -p PID -F`.
- Categorize FDs: regular files, sockets (TCP/UDP/Unix), pipes, devices, other.

**Display:**
- Detail view: new tab/section activated by pressing `f` (file descriptors)
- Categorized list: "Files: 12  Sockets: 8  Pipes: 3"
- Each socket shows: `TCP 192.168.1.5:443 → 10.0.0.1:52341 (ESTABLISHED)`
- Each file shows: `/var/log/app.log (read)`
- Scrollable with j/k within the FD list

**Replaces:** `lsof -p PID`

**Effort:** Medium — Linux is straightforward via procfs. macOS requires cgo or
shelling out.

---

### 4. Disk I/O Per Process

**What htop lacks:** htop has a basic I/O column but no rates, no history,
no sparklines. `iotop` is a separate tool requiring root.

**Implementation approach:**
- Linux: read `/proc/PID/io` for `read_bytes` and `write_bytes`. Calculate
  delta between snapshots for rate (bytes/sec).
- macOS: `proc_pidinfo` with `PROC_PIDTASKINFO` via cgo, or `fs_usage`
  parsing (requires root).
- Fallback: show "N/A" when data is unavailable.

**Display:**
- New columns: `IO-R` (read rate), `IO-W` (write rate) — toggleable with `d`
  key to avoid table clutter
- Detail view: disk I/O sparkline history (read + write)
- Sort by I/O rate: add `SortByIORead`, `SortByIOWrite` to sort cycle

**Replaces:** `iotop`

**Effort:** Medium — `/proc/PID/io` is simple. Rate calculation reuses existing
`Collector` delta pattern.

---

### 5. Process Timeline / Historical Replay

**What htop lacks:** htop is purely real-time. Close it and all data is gone.
"What was using CPU 3 minutes ago?" is unanswerable.

**Implementation approach:**
- Maintain a rolling circular buffer of full `Snapshot` structs in memory.
  At 2-second intervals, 300 snapshots = 10 minutes of history.
- Memory budget: each snapshot is ~25 processes × ~200 bytes + system metrics
  ≈ 6 KiB per snapshot. 300 × 6 KiB ≈ 1.8 MiB — acceptable.
- Timeline mode: press `t` to enter, `[` / `]` to step backward/forward,
  `Esc` to return to live view.
- In timeline mode, all panels render from the historical snapshot instead
  of the live one. Header shows "REPLAY: 3m 24s ago" with a visual indicator.

**Display:**
- Timeline bar at top: `[|||||||||||||●|||||] -3m24s` showing position
- All existing panels work unchanged (they take a `Snapshot` — just pass
  the historical one)
- Process table is interactive in replay mode (cursor, sort still work)
- Enter on a process in replay shows detail with "snapshot at T-3m24s" label

**Replaces:** Nothing — no terminal monitor offers this.

**Effort:** Medium — data structure is simple (ring buffer of Snapshots). Main
work is the timeline navigation UI and the replay indicator rendering.

---

## Tier 2: Moderate Effort, Strong Differentiation

### 6. GPU Monitoring Panel

**What htop/top lacks:** No GPU visibility at all. Users run `nvidia-smi` or
`intel_gpu_top` separately.

**Implementation approach:**
- NVIDIA: parse `nvidia-smi --query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu --format=csv,noheader,nounits`
  or use NVML bindings via cgo.
- AMD: read ROCm sysfs files (`/sys/class/drm/card*/device/gpu_busy_percent`).
- Apple Silicon: IOKit GPU utilization via `powermetrics` parsing or
  `IOServiceGetMatchingServices` via cgo.
- Intel: `intel_gpu_top` parsing or i915 perf counters.

**Display:**
- New summary line: `GPU [████░░░░░░] 45%  VRAM: 4.2/8.0 GiB  Temp: 72°C`
- Per-process GPU% column (NVIDIA only via `nvidia-smi pmon`)
- Detail view: GPU history sparkline for the selected process

**Replaces:** `nvidia-smi`, `intel_gpu_top`, `radeontop`

**Effort:** High — multi-vendor support is complex. Start with NVIDIA only.

---

### 7. Process Grouping by Application

**What htop lacks:** Chrome spawns 40+ processes. htop shows all 40 as
separate rows. Users can't see the aggregate resource footprint of an
application at a glance.

**Implementation approach:**
- Group by process name: aggregate CPU%, MEM%, RSS, CONN for all PIDs with
  the same name.
- Data structure: `map[string]ProcessGroup` where `ProcessGroup` holds
  aggregated metrics + list of member PIDs.
- Toggle with `p` key: grouped view ↔ flat view.
- Expand/collapse groups with `+`/`-` or `Enter`.

**Display:**
- Grouped row: `chrome (38)   brajesh  12.4   8.2  3.1G  147`
  where `(38)` is process count.
- Expanded: indented child rows underneath the group header.
- Sort applies to group aggregates in grouped mode.
- Detail view on a group: shows aggregate sparklines + member PID list.

**Replaces:** Nothing — unique capability.

**Effort:** Medium — sorting and cursor management become more complex with
nested rows. Need a tree-like data structure for the process list.

---

### 8. Threshold Alerts with Visual Flash

**What htop lacks:** No alerting. Users must visually scan for problems.

**Implementation approach:**
- Default thresholds: CPU% > 90, MEM% > 85, Disk > 95%, single process > 50% CPU.
- Optional config file: `~/.config/termdash/alerts.yaml`
  ```yaml
  alerts:
    - metric: cpu_total
      threshold: 90
      message: "CPU critical"
    - metric: process_cpu
      threshold: 50
      process_name: ".*"
    - metric: disk_percent
      threshold: 95
      mount: "/"
  ```
- Alert state tracked in model: list of active alerts with timestamps.
- Visual: affected rows/bars flash between normal and danger color (toggle
  every render cycle). Alert banner at bottom above help line.

**Display:**
- Alert banner: `⚠ ALERT: CPU total 94% | chrome CPU 52% | / disk 96%`
- Flashing row highlight for processes exceeding thresholds.
- Press `a` to view alert history (last 50 alerts with timestamps).

**Replaces:** Custom monitoring scripts.

**Effort:** Medium — config parsing + alert state tracking + visual flashing.

---

### 9. Snapshot Export (JSON / CSV)

**What htop lacks:** Cannot export data without screen scraping.

**Implementation approach:**
- Press `e` to dump current `Snapshot` + `connCounts` to
  `~/termdash-{timestamp}.json`.
- Press `E` to toggle continuous CSV logging to
  `~/termdash-log-{timestamp}.csv` (one row per process per snapshot).
- JSON schema matches the `Snapshot` struct directly via `encoding/json`.
- CSV columns: `timestamp,pid,name,user,cpu_pct,mem_pct,rss,conn_count`.

**Display:**
- Status line briefly shows: "Exported to ~/termdash-2025-01-15T14:32:05.json"
- During continuous logging: indicator in header "REC●" in red.

**Replaces:** Nothing — unique capability. Enables feeding data into Grafana,
spreadsheets, or custom analysis scripts.

**Effort:** Low — serialization is straightforward. File I/O in a tea.Cmd.

---

### 10. Process Communication Map

**What htop lacks:** No visibility into inter-process communication.

**Implementation approach:**
- From the connection table (already collected), identify connections where
  both endpoints are local PIDs.
- Build an adjacency list: `PID-A → PID-B via TCP:3000`.
- Also detect Unix socket communication via `/proc/net/unix` inode matching.
- Resolve PIDs to process names.

**Display:**
- Detail view: new section "Communicates with:"
  ```
  → node (PID 456) via TCP :3000
  → postgres (PID 789) via TCP :5432
  ← nginx (PID 123) via TCP :8080
  ```
- Direction: `→` outbound, `←` inbound.
- Press `m` in dashboard for a full-screen communication map:
  ```
  nginx:80 → node:3000 → postgres:5432
                       → redis:6379
  ```

**Replaces:** Manual `lsof` + `ss` cross-referencing.

**Effort:** Medium — socket inode resolution is the hard part. Display is
straightforward text.

---

## Tier 3: Aspirational (High Effort, Very Unique)

### 11. Smart Anomaly Detection

**What no terminal monitor has:** Statistical anomaly detection for process
resource usage.

**Implementation approach:**
- For each process name, maintain a rolling 5-minute exponential moving average
  (EMA) of CPU% and MEM%.
- Flag a process when current value > EMA + 2× standard deviation.
- Use a simple online algorithm (Welford's) for running mean and variance.
- Memory: ~100 bytes per process name (mean, variance, count). For 200 unique
  names: 20 KiB.

**Display:**
- `!` indicator next to process name in table for anomalous processes.
- Tooltip on detail view: "CPU 340% above 5-min average (normal: 2.1%)"
- Press `!` in dashboard to filter to anomalous processes only.

**Replaces:** Nothing — entirely novel for terminal tools.

**Effort:** Medium-High — statistical tracking is simple, but tuning thresholds
to avoid false positives requires testing.

---

### 12. Integrated Log Tail

**What htop lacks:** Must switch to `journalctl` or `tail -f` in another
terminal.

**Implementation approach:**
- Linux: for systemd services, query journal via `journalctl -u {service} -n 50 --no-pager`.
  Map PID → systemd unit via `/proc/PID/cgroup`.
- Generic: read `/proc/PID/fd/1` (stdout) and `/proc/PID/fd/2` (stderr) if
  they point to regular files or ptys.
- Docker: `docker logs --tail 50 {container_id}` for containerized processes.

**Display:**
- Detail view: press `l` to open log panel (bottom half of detail view).
- Scrollable log with timestamps.
- Log auto-scrolls unless user scrolls up (sticky bottom).
- Syntax highlighting for common patterns: ERROR (red), WARN (amber),
  timestamps (muted).

**Replaces:** `journalctl -f`, `docker logs -f`, `tail -f`

**Effort:** High — log source detection is complex. Stdout/stderr reading
has race conditions and permission issues.

---

### 13. Remote Mode via SSH

**What htop lacks:** htop must be installed on the target host. No remote
monitoring from a single machine.

**Implementation approach:**
- `termdash ssh user@host` — establish SSH connection, transfer a minimal
  metrics-collection binary (or run Go commands via SSH), stream JSON snapshots
  back over the SSH channel.
- Architecture: collector runs remotely (embedded or via `go run`), UI runs
  locally. Protocol: newline-delimited JSON over SSH stdin/stdout.
- Alternative: transfer the `termdash` binary itself to `/tmp/` on the remote
  host, execute it in "headless collector" mode (`termdash --collect-only`),
  pipe output back.

**Display:**
- Header shows: `⣿ termdash [remote: user@host]`
- All features work identically — the UI only needs a `Snapshot` struct.
- Connection status indicator: green dot = connected, red dot = disconnected.

**Replaces:** SSH + htop installation on every server.

**Effort:** High — SSH session management, binary transfer, error handling for
network interruptions.

---

## Prioritized Implementation Order

Recommended order based on impact-to-effort ratio and "wow factor" for
differentiating termdash from htop:

| Priority | Feature | Effort | Replaces | Unique? |
|----------|---------|--------|----------|---------|
| 1 | Process grouping by app (#7) | Medium | — | Yes |
| 2 | Historical replay (#5) | Medium | — | Yes |
| 3 | Snapshot export JSON/CSV (#9) | Low | — | Yes |
| 4 | Container column (#2) | Low-Med | docker stats | Partially |
| 5 | Disk I/O per process (#4) | Medium | iotop | Partially |
| 6 | Open file inspector (#3) | Medium | lsof | Partially |
| 7 | Per-process net bandwidth (#1) | High | nethogs | Partially |
| 8 | Threshold alerts (#8) | Medium | custom scripts | Yes |
| 9 | Anomaly detection (#11) | Med-High | — | Yes |
| 10 | Process comm map (#10) | Medium | manual lsof+ss | Yes |
| 11 | GPU monitoring (#6) | High | nvidia-smi | Partially |
| 12 | Integrated log tail (#12) | High | journalctl | Partially |
| 13 | Remote SSH mode (#13) | High | ssh+htop | Partially |

### The Elevator Pitch After Implementing 1-5:

> "termdash: htop with application grouping, 10-minute replay,
> container awareness, disk I/O per process, and one-key JSON export.
> Single binary, zero config."

### The Full Vision After All 13:

> "termdash: the last system monitor you'll ever need. Replaces htop +
> nethogs + iotop + lsof + docker stats + nvidia-smi in a single binary
> with historical replay, anomaly detection, and remote SSH monitoring."
