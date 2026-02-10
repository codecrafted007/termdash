# TermDash Configuration Language Specification

**Version:** 0.1.0
**File extension:** `.td`
**Default path:** `~/.config/termdash/config.td`

---

## 1. Formal Grammar (BNF-like)

```
config          = { statement }

statement       = global_block
                | theme_block
                | layout_block
                | widget_block
                | metric_block
                | alert_block
                | source_block
                | style_block

# ── Global settings ──────────────────────────────────────────────

global_block    = "global" "{" { global_attr } "}"
global_attr     = IDENT "=" expr

# ── Theme ────────────────────────────────────────────────────────

theme_block     = "theme" [ STRING ] "{" { theme_attr } "}"
theme_attr      = IDENT "=" expr

# ── Layout ───────────────────────────────────────────────────────

layout_block    = "layout" "{" { layout_row } "}"
layout_row      = "row" [ "weight" NUMBER ] "{" { layout_cell } "}"
layout_cell     = "col" [ "weight" NUMBER ] "{" widget_ref "}"
                | "col" [ "weight" NUMBER ] "{" { layout_row } "}"
widget_ref      = IDENT [ "(" [ arg_list ] ")" ]

# ── Widget ───────────────────────────────────────────────────────

widget_block    = "widget" STRING "{" { widget_attr | when_block | style_block } "}"
widget_attr     = IDENT "=" expr
                | "filter" "=" expr
                | "columns" "=" "[" ident_list "]"
                | "sort" "=" IDENT [ "asc" | "desc" ]

# ── Computed metrics ─────────────────────────────────────────────

metric_block    = "metric" STRING "{" { metric_attr } "}"
metric_attr     = "expr" "=" expr
                | "unit" "=" STRING
                | "format" "=" STRING
                | "window" "=" duration

# ── Alert rules ──────────────────────────────────────────────────

alert_block     = "alert" STRING "{" { alert_attr } "}"
alert_attr      = "when" "=" expr
                | "for" "=" duration
                | "message" "=" expr
                | "severity" "=" ( "info" | "warn" | "critical" )
                | "action" action_block

action_block    = "{" { action_attr } "}"
action_attr     = "flash" "=" BOOL
                | "exec" "=" STRING
                | "notify" "=" BOOL
                | "sound" "=" BOOL

# ── Custom data sources ─────────────────────────────────────────

source_block    = "source" STRING "{" { source_attr } "}"
source_attr     = "type" "=" ( "exec" | "file" | "http" )
                | "command" "=" STRING
                | "path" "=" STRING
                | "url" "=" STRING
                | "interval" "=" duration
                | "format" "=" ( "number" | "text" | "json" )
                | "json_path" "=" STRING
                | "timeout" "=" duration
                | "env" "=" "{" { IDENT "=" STRING } "}"

# ── Conditional blocks ──────────────────────────────────────────

when_block      = "when" expr "{" { widget_attr | style_attr } "}"

# ── Style rules ──────────────────────────────────────────────────

style_block     = "style" [ STRING ] "{" { style_attr | when_block } "}"
style_attr      = "fg" "=" color_expr
                | "bg" "=" color_expr
                | "bold" "=" BOOL
                | "dim" "=" BOOL
                | "border" "=" ( "rounded" | "thick" | "double" | "none" )
                | "border_fg" "=" color_expr
                | "sparkline" "=" color_expr
                | "label" "=" color_expr
                | "value" "=" color_expr
                | "bar" "=" color_expr

# ── Expressions ──────────────────────────────────────────────────

expr            = or_expr
or_expr         = and_expr { "or" and_expr }
and_expr        = not_expr { "and" not_expr }
not_expr        = [ "not" ] compare_expr
compare_expr    = add_expr { cmp_op add_expr }
add_expr        = mul_expr { ( "+" | "-" ) mul_expr }
mul_expr        = unary_expr { ( "*" | "/" | "%" ) unary_expr }
unary_expr      = [ "-" ] postfix_expr
postfix_expr    = primary { "." IDENT | "(" [ arg_list ] ")" }
primary         = NUMBER
                | STRING
                | BOOL
                | IDENT
                | duration
                | "(" expr ")"

cmp_op          = ">" | "<" | ">=" | "<=" | "=" | "!=" | "~"
arg_list        = expr { "," expr }
ident_list      = IDENT { "," IDENT }
color_expr      = STRING                        # hex: "#FF6B6B"
                | IDENT                         # named: red, green, amber
                | "gradient" "(" expr "," color_expr "," color_expr ")"

# ── Lexical elements ────────────────────────────────────────────

IDENT           = letter { letter | digit | "_" }
NUMBER          = digit { digit } [ "." digit { digit } ]
STRING          = '"' { char } '"'
BOOL            = "true" | "false"
duration        = NUMBER ( "ms" | "s" | "m" | "h" )
COMMENT         = "#" { any } newline
```

---

## 2. Example Configuration Files

### 2a. Minimal Config -- Customize Layout

```termdash
# ~/.config/termdash/config.td
# Minimal: just rearrange the default widgets with a custom refresh rate.

global {
    refresh = 1s
    title   = "my workstation"
}

layout {
    # Top row: CPU and memory side by side, CPU gets 60% of the width
    row weight 1 {
        col weight 3 { cpu }
        col weight 2 { memory }
    }

    # Middle row: disk and network, equal width
    row weight 1 {
        col weight 1 { disk }
        col weight 1 { network }
    }

    # Bottom row: process table takes all available space
    row weight 3 {
        col weight 1 { procs }
    }
}
```

### 2b. Power User Config -- Computed Metrics, Alerts, Custom Sources

```termdash
# ~/.config/termdash/config.td
# Power user: derived metrics, alerting, and custom data sources.

global {
    refresh  = 2s
    title    = "dev workstation"
    history  = 10m       # keep 10 minutes of replay data
    log_path = "~/.local/share/termdash/metrics.csv"
}

theme {
    primary   = "#7D56F4"
    success   = "#04B575"
    warning   = "#FFBE0B"
    danger    = "#FF6B6B"
    bg        = "#1A1B26"
    fg        = "#C0CAF5"
}

# ── Computed metrics ─────────────────────────────────────────────

metric "mem_pressure" {
    expr   = memory.ram_percent * 0.7 + memory.swap_percent * 0.3
    unit   = "%"
    format = "%.1f"
}

metric "net_total" {
    expr   = network.send_rate + network.recv_rate
    unit   = "B/s"
    format = "auto"    # auto-scales to KiB/s, MiB/s, etc.
}

metric "cpu_io_ratio" {
    expr   = cpu.total / max(net_total, 1)
    unit   = ""
    format = "%.2f"
}

# ── Custom data sources ─────────────────────────────────────────

source "docker_count" {
    type     = exec
    command  = "docker ps -q | wc -l"
    interval = 10s
    format   = number
}

source "go_routines" {
    type      = http
    url       = "http://localhost:6060/debug/vars"
    interval  = 5s
    format    = json
    json_path = "goroutines"
    timeout   = 2s
}

source "tailscale_status" {
    type     = exec
    command  = "tailscale status --json | jq -r '.Self.Online'"
    interval = 30s
    format   = text
}

source "load_avg" {
    type   = file
    path   = "/proc/loadavg"
    format = text
}

# ── Layout ───────────────────────────────────────────────────────

layout {
    row weight 1 {
        col weight 3 {
            cpu(
                show_per_core = true
                sparkline     = true
            )
        }
        col weight 2 {
            memory(sparkline = true)
        }
    }

    row weight 1 {
        col weight 1 { disk }
        col weight 1 { network(sparkline = true) }
        col weight 1 {
            # Custom status widget showing data from sources
            widget "status" {
                type = table
                rows = [
                    ["Docker",    docker_count]
                    ["Goroutines", go_routines]
                    ["Tailscale", tailscale_status]
                    ["Load",      load_avg]
                ]
            }
        }
    }

    row weight 3 {
        col weight 1 {
            procs(
                columns = [pid, name, user, cpu, mem, conn, io_r, io_w]
                sort    = cpu desc
                filter  = cpu > 0.1 or conn > 0
            )
        }
    }
}

# ── Alert rules ──────────────────────────────────────────────────

alert "cpu_critical" {
    when     = cpu.total > 90
    for      = 30s                # must sustain for 30 seconds
    message  = "CPU at {cpu.total}% for 30s"
    severity = critical
    action {
        flash  = true
        notify = true             # OS notification
    }
}

alert "mem_pressure_high" {
    when     = mem_pressure > 80
    for      = 1m
    message  = "Memory pressure at {mem_pressure}%"
    severity = warn
    action {
        flash = true
    }
}

alert "disk_full" {
    when     = disk.root.percent > 95
    for      = 0s                  # trigger immediately
    message  = "Root disk {disk.root.percent}% full"
    severity = critical
    action {
        flash  = true
        notify = true
        exec   = "logger -t termdash 'disk alert: root at {disk.root.percent}%'"
    }
}

alert "runaway_process" {
    when     = procs.any(cpu > 80 and name != "gcc" and name != "cargo")
    for      = 2m
    message  = "Process {name} using {cpu}% CPU for 2+ minutes"
    severity = warn
    action {
        flash = true
    }
}

# ── Conditional styling ─────────────────────────────────────────

style "cpu_bar" {
    when cpu.total < 50 {
        bar = "#04B575"          # green
    }
    when cpu.total >= 50 and cpu.total < 80 {
        bar = "#FFBE0B"          # amber
    }
    when cpu.total >= 80 {
        bar = "#FF6B6B"          # red
        bold = true
    }
}

style "process_row" {
    # Highlight processes with high CPU in the table
    when cpu > 50 {
        fg   = "#FF6B6B"
        bold = true
    }
    when mem > 10 {
        fg = "#FFBE0B"
    }
    when conn > 100 {
        fg = "#FF8C00"
    }
}
```

### 2c. Server Monitoring Config -- Services, Network, Disk I/O

```termdash
# /etc/termdash/server.td
# Server monitoring: focus on services, disk I/O, and network throughput.

global {
    refresh  = 3s
    title    = "prod-api-01"
    history  = 30m
    mouse    = false          # disable mouse in server terminals
}

theme {
    primary = "#61AFEF"       # blue-ish for server look
    bg      = "#282C34"
}

# ── Custom data sources for service health ───────────────────────

source "nginx_connections" {
    type      = http
    url       = "http://127.0.0.1:8080/stub_status"
    interval  = 5s
    format    = text
    timeout   = 2s
}

source "postgres_connections" {
    type     = exec
    command  = "psql -t -c 'SELECT count(*) FROM pg_stat_activity'"
    interval = 10s
    format   = number
    env {
        PGHOST     = "localhost"
        PGUSER     = "monitor"
        PGDATABASE = "postgres"
    }
}

source "redis_mem" {
    type     = exec
    command  = "redis-cli info memory | grep used_memory_human | cut -d: -f2"
    interval = 10s
    format   = text
}

source "queue_depth" {
    type      = http
    url       = "http://localhost:15672/api/queues/%2F/tasks"
    interval  = 5s
    format    = json
    json_path = "messages"
    timeout   = 3s
}

source "cert_days" {
    type     = exec
    command  = "echo | openssl s_client -connect localhost:443 2>/dev/null | openssl x509 -noout -enddate | cut -d= -f2 | xargs -I{} python3 -c \"from datetime import datetime; print((datetime.strptime('{}', '%b %d %H:%M:%S %Y %Z')-datetime.now()).days)\""
    interval = 1h
    format   = number
}

# ── Computed metrics ─────────────────────────────────────────────

metric "disk_io_total" {
    expr   = disk.sda.read_rate + disk.sda.write_rate
    unit   = "B/s"
    format = "auto"
}

metric "net_utilization" {
    # Assuming 1 Gbps link
    expr   = (network.send_rate + network.recv_rate) / 125000000 * 100
    unit   = "%"
    format = "%.1f"
}

metric "service_health" {
    # 1 = all healthy, 0 = something is down
    expr = min(
        postgres_connections > 0,
        nginx_connections > 0,
        queue_depth < 10000
    )
    format = "%.0f"
}

# ── Layout ───────────────────────────────────────────────────────

layout {
    # Top: system overview in a compact row
    row weight 1 {
        col weight 1 { cpu(sparkline = true, show_per_core = false) }
        col weight 1 { memory(sparkline = true) }
        col weight 1 { network(sparkline = true) }
    }

    # Middle: disk and services
    row weight 1 {
        col weight 2 {
            disk(
                show_io     = true
                mounts      = ["/", "/data", "/var/log"]
                sparkline   = true
            )
        }
        col weight 1 {
            widget "services" {
                type  = table
                title = "Service Health"
                rows  = [
                    ["Nginx",    nginx_connections, "conns"]
                    ["Postgres", postgres_connections, "conns"]
                    ["Redis",    redis_mem]
                    ["Queue",    queue_depth, "msgs"]
                    ["TLS Cert", cert_days, "days"]
                ]

                # Conditional styling within the widget
                when queue_depth > 5000 {
                    fg = "#FFBE0B"
                }
                when queue_depth > 10000 {
                    fg = "#FF6B6B"
                }
                when cert_days < 14 {
                    fg = "#FF6B6B"
                    bold = true
                }
            }
        }
    }

    # Bottom: process table filtered to interesting server processes
    row weight 3 {
        col weight 1 {
            procs(
                columns  = [pid, name, user, cpu, mem, io_r, io_w, conn]
                sort     = cpu desc
                filter   = name ~ "nginx" or name ~ "postgres"
                          or name ~ "redis" or name ~ "node"
                          or name ~ "python" or cpu > 5
                group_by = name
            )
        }
    }
}

# ── Alert rules ──────────────────────────────────────────────────

alert "high_load" {
    when     = cpu.total > 85
    for      = 1m
    message  = "CPU sustained at {cpu.total}%"
    severity = critical
    action {
        flash  = true
        notify = true
        exec   = "curl -s -X POST https://hooks.slack.com/trigger/xxx -d '{\"text\": \"prod-api-01 CPU {cpu.total}%\"}'"
    }
}

alert "memory_exhaustion" {
    when     = memory.ram_percent > 90
    for      = 30s
    message  = "RAM at {memory.ram_percent}%, swap at {memory.swap_percent}%"
    severity = critical
    action {
        flash  = true
        notify = true
    }
}

alert "disk_space" {
    when     = disk.root.percent > 85
    for      = 0s
    message  = "Disk / at {disk.root.percent}%"
    severity = warn
    action {
        flash = true
    }
}

alert "disk_io_saturated" {
    when     = disk_io_total > 500000000    # 500 MB/s
    for      = 15s
    message  = "Disk I/O at {disk_io_total} -- possible thrashing"
    severity = warn
    action {
        flash = true
    }
}

alert "queue_backlog" {
    when     = queue_depth > 10000
    for      = 5m
    message  = "Queue depth at {queue_depth} for 5+ minutes"
    severity = critical
    action {
        flash  = true
        notify = true
        exec   = "curl -s -X POST https://hooks.slack.com/trigger/xxx -d '{\"text\": \"Queue backlog: {queue_depth}\"}'"
    }
}

alert "cert_expiring" {
    when     = cert_days < 14
    for      = 0s
    message  = "TLS certificate expires in {cert_days} days"
    severity = warn
    action {
        notify = true
    }
}

alert "postgres_maxed" {
    when     = postgres_connections > 90
    for      = 30s
    message  = "PostgreSQL at {postgres_connections} connections"
    severity = warn
    action {
        flash  = true
        notify = true
    }
}

alert "network_saturated" {
    when     = net_utilization > 80
    for      = 1m
    message  = "Network at {net_utilization}% of 1Gbps capacity"
    severity = warn
    action {
        flash = true
    }
}

# ── Styling ──────────────────────────────────────────────────────

style "process_row" {
    when cpu > 30 {
        fg   = "#FF6B6B"
        bold = true
    }
    when conn > 50 {
        fg = "#FF8C00"
    }
}

style "disk_bar" {
    when disk.root.percent < 70 {
        bar = "#04B575"
    }
    when disk.root.percent >= 70 and disk.root.percent < 90 {
        bar = "#FFBE0B"
    }
    when disk.root.percent >= 90 {
        bar = "#FF6B6B"
    }
}
```

---

## 3. Complete Keyword and Operator Reference

### 3.1 Block Keywords

| Keyword    | Purpose                                          |
|------------|--------------------------------------------------|
| `global`   | Top-level settings: refresh, title, history      |
| `theme`    | Color palette definition                         |
| `layout`   | Widget arrangement in rows and columns           |
| `row`      | Horizontal row within a layout                   |
| `col`      | Column within a row                              |
| `widget`   | Inline widget declaration (inside a layout col)  |
| `metric`   | User-defined computed metric                     |
| `alert`    | Threshold-based alert rule                       |
| `source`   | Custom external data source                      |
| `style`    | Conditional styling rules                        |
| `when`     | Conditional block (inside widget or style)       |
| `action`   | Alert response configuration                     |

### 3.2 Built-in Widget Identifiers

| Identifier | Maps to                          | Configurable Options                         |
|------------|----------------------------------|----------------------------------------------|
| `cpu`      | CPU usage panel                  | `show_per_core`, `sparkline`                 |
| `memory`   | RAM + Swap panel                 | `sparkline`                                  |
| `disk`     | Disk usage panel                 | `mounts`, `show_io`, `sparkline`             |
| `network`  | Network throughput panel         | `sparkline`                                  |
| `procs`    | Process table                    | `columns`, `sort`, `filter`, `group_by`      |
| `gpu`      | GPU utilization panel (future)   | `vendor`, `sparkline`                        |

### 3.3 Attribute Keywords

| Keyword      | Context          | Type       | Description                             |
|--------------|------------------|------------|-----------------------------------------|
| `refresh`    | global           | duration   | Data collection interval                |
| `title`      | global, widget   | string     | Display title                           |
| `history`    | global           | duration   | Replay buffer duration                  |
| `log_path`   | global           | string     | Path for CSV auto-logging               |
| `mouse`      | global           | bool       | Enable/disable mouse support            |
| `weight`     | row, col         | number     | Proportional size (relative to siblings)|
| `type`       | source, widget   | ident      | Source type or widget type               |
| `command`    | source           | string     | Shell command for exec source           |
| `path`       | source           | string     | File path for file source               |
| `url`        | source           | string     | HTTP endpoint for http source           |
| `interval`   | source           | duration   | Poll interval for source                |
| `format`     | source, metric   | string/id  | Output format (number/text/json/auto)   |
| `json_path`  | source           | string     | JSONPath for extracting values           |
| `timeout`    | source           | duration   | Request timeout                         |
| `env`        | source           | block      | Environment variables for exec          |
| `expr`       | metric           | expression | Computation expression                  |
| `unit`       | metric           | string     | Display unit suffix                     |
| `window`     | metric           | duration   | Rolling window for aggregations         |
| `severity`   | alert            | ident      | Alert level: info, warn, critical       |
| `message`    | alert            | string     | Alert message (supports interpolation)  |
| `for`        | alert            | duration   | Sustained duration before firing        |
| `flash`      | action           | bool       | Visual flash on alert                   |
| `exec`       | action           | string     | Shell command to run on alert           |
| `notify`     | action           | bool       | Send OS notification                    |
| `sound`      | action           | bool       | Play alert sound                        |
| `columns`    | procs widget     | list       | Visible columns                         |
| `sort`       | procs widget     | ident+dir  | Sort column and direction               |
| `filter`     | procs widget     | expression | Process filter (superset of query DSL)  |
| `group_by`   | procs widget     | ident      | Group processes by field                |
| `mounts`     | disk widget      | list       | Mount points to display                 |
| `show_io`    | disk widget      | bool       | Show read/write rates                   |
| `show_per_core` | cpu widget   | bool       | Show per-core breakdown                 |
| `sparkline`  | any widget       | bool       | Show sparkline history                  |
| `rows`       | table widget     | list       | Row definitions for custom table        |

### 3.4 Style Attribute Keywords

| Keyword      | Type        | Description                              |
|--------------|-------------|------------------------------------------|
| `fg`         | color       | Foreground text color                    |
| `bg`         | color       | Background color                         |
| `bold`       | bool        | Bold text                                |
| `dim`        | bool        | Dimmed text                              |
| `border`     | ident       | Border style (rounded, thick, double, none) |
| `border_fg`  | color       | Border color                             |
| `bar`        | color       | Bar/sparkline fill color                 |
| `label`      | color       | Label text color                         |
| `value`      | color       | Value text color                         |

### 3.5 Comparison Operators

| Operator | Meaning                | Precedence | Notes                              |
|----------|------------------------|------------|------------------------------------|
| `=`      | Equals                 | 4          | Single `=` for comparison (not `==`) |
| `!=`     | Not equals             | 4          |                                    |
| `>`      | Greater than           | 4          |                                    |
| `<`      | Less than              | 4          |                                    |
| `>=`     | Greater than or equal  | 4          |                                    |
| `<=`     | Less than or equal     | 4          |                                    |
| `~`      | Contains (strings)     | 4          | Case-insensitive substring match   |

### 3.6 Arithmetic Operators

| Operator | Meaning        | Precedence |
|----------|----------------|------------|
| `+`      | Addition       | 5          |
| `-`      | Subtraction    | 5          |
| `*`      | Multiplication | 6          |
| `/`      | Division       | 6          |
| `%`      | Modulo         | 6          |
| `-`      | Unary negation | 7          |

### 3.7 Boolean Operators

| Operator | Meaning        | Precedence | Notes                               |
|----------|----------------|------------|---------------------------------------|
| `and`    | Logical AND    | 2          | Word-based, not `&&`                  |
| `or`     | Logical OR     | 1          | Word-based, not `\|\|`               |
| `not`    | Logical NOT    | 3          | Prefix operator                       |

### 3.8 Other Operators and Syntax

| Syntax            | Meaning                                    |
|-------------------|--------------------------------------------|
| `.`               | Field access: `cpu.total`, `disk.root.percent` |
| `()`              | Function call or widget options             |
| `[]`              | List literal                                |
| `{}`              | Block delimiter                             |
| `#`               | Line comment (to end of line)               |
| `"..."`           | String literal                              |
| `{name}`          | String interpolation (inside message strings) |

### 3.9 Duration Literals

| Suffix | Meaning       | Example  |
|--------|---------------|----------|
| `ms`   | Milliseconds  | `500ms`  |
| `s`    | Seconds       | `30s`    |
| `m`    | Minutes       | `10m`    |
| `h`    | Hours         | `1h`     |

### 3.10 Built-in Functions

| Function                  | Description                                        |
|---------------------------|----------------------------------------------------|
| `max(a, b)`               | Returns the larger of two values                   |
| `min(a, b)`               | Returns the smaller of two values                  |
| `avg(metric, window)`     | Rolling average over a time window                 |
| `sum(metric, window)`     | Rolling sum over a time window                     |
| `rate(metric)`            | Rate of change per second                          |
| `abs(x)`                  | Absolute value                                     |
| `ceil(x)`                 | Ceiling                                            |
| `floor(x)`                | Floor                                              |
| `round(x, decimals)`      | Round to N decimal places                          |
| `clamp(x, low, high)`     | Clamp value to range                               |
| `gradient(value, c1, c2)` | Interpolate color between c1 and c2 based on value |
| `procs.any(expr)`         | True if any process matches expr                   |
| `procs.count(expr)`       | Count of processes matching expr                   |

### 3.11 Built-in Metric Namespace

These are the dotted paths available for use in expressions, derived from the
existing `metrics.Snapshot` struct:

```
cpu.total                   # float64, 0-100
cpu.core[N]                 # float64, per-core (0-indexed)

memory.ram_percent          # float64, 0-100
memory.swap_percent         # float64, 0-100
memory.used_ram             # uint64, bytes
memory.total_ram            # uint64, bytes
memory.used_swap            # uint64, bytes
memory.total_swap           # uint64, bytes

disk.<mount>.percent        # float64, 0-100 (mount aliased: "/" -> "root")
disk.<mount>.used           # uint64, bytes
disk.<mount>.total          # uint64, bytes
disk.<mount>.read_rate      # float64, bytes/sec (future)
disk.<mount>.write_rate     # float64, bytes/sec (future)

network.send_rate           # float64, bytes/sec
network.recv_rate           # float64, bytes/sec
network.bytes_sent          # uint64, cumulative
network.bytes_recv          # uint64, cumulative

# Per-process fields (available inside filter, when, alert with procs.any):
name                        # string
user                        # string
pid                         # int32
cpu                         # float64, 0-100+
mem                         # float32, 0-100
conn                        # int, connection count
io_r                        # float64, bytes/sec read (future)
io_w                        # float64, bytes/sec write (future)

# Custom sources are referenced by their declared name:
<source_name>               # value from the source
```

### 3.12 Named Colors

| Name       | Hex       | Usage                    |
|------------|-----------|--------------------------|
| `red`      | `#FF6B6B` | Danger, critical alerts  |
| `green`    | `#04B575` | Success, healthy         |
| `amber`    | `#FFBE0B` | Warning                  |
| `blue`     | `#61AFEF` | Informational            |
| `purple`   | `#7D56F4` | Primary/accent           |
| `indigo`   | `#6C63FF` | Secondary                |
| `orange`   | `#FF8C00` | Connections              |
| `gray`     | `#626262` | Muted/dim                |
| `white`    | `#FAFAFA` | Primary text             |

---

## 4. Design Tradeoffs and Decisions

### 4.1 Single `=` for Comparison, Not `==`

**Decision:** Use `=` for equality comparisons (`cpu.total = 50`), not `==`.

**Rationale:** The existing query DSL already uses `=` for equality
(`name = chrome`, `cpu > 50`). Since this config language must be a superset of
that syntax, we preserve it. There is no variable assignment in expressions
(attributes use `=` in a key-value context where the left-hand side is always a
bare keyword, never an expression), so there is no ambiguity. This matches SQL
convention and reduces punctuation.

**Tradeoff:** Users coming from C/Python/Go may instinctively write `==`. The
parser should accept `==` as a synonym for `=` in expression contexts to be
forgiving, but canonical style uses single `=`.

### 4.2 Word-Based Boolean Operators (`and`, `or`, `not`)

**Decision:** Use English words, not `&&`, `||`, `!`.

**Rationale:** The existing query parser already uses `and`/`or`
(case-insensitive). The filter expression `cpu > 5 and name ~ "chrome"` reads
naturally. Symbolic operators add visual noise and make the config less readable
for non-programmers. This matches SQL and HCL conventions.

**Tradeoff:** `not` as a prefix keyword reads slightly less naturally than `!`
in complex expressions (`not (cpu > 50 and mem > 80)` vs `!(cpu > 50 && mem > 80)`).
The verbosity is a net win for readability.

### 4.3 Braces `{}` for Blocks, Not Indentation

**Decision:** Curly braces delimit all blocks.

**Rationale:** Brace-delimited blocks are easier to parse (no indentation
tracking), compose well in nested structures (layout rows inside cols), and
avoid whitespace ambiguity when configs are copied between systems, editors,
or documentation. This matches HCL/Terraform convention.

**Tradeoff:** Slightly more visual noise than pure indentation-based syntax.
Mitigated by the fact that most blocks are short (2-5 attributes).

### 4.4 Duration Literals as Suffixed Numbers

**Decision:** Durations are expressed as `30s`, `5m`, `1h`, not as strings
(`"30s"`) or separate key-value pairs (`seconds = 30`).

**Rationale:** Duration literals are first-class values that appear in three
different contexts (global refresh, alert `for`, source interval). Suffixed
number syntax is immediately readable (`for = 30s`) and eliminates parsing
ambiguity. This matches Prometheus and Grafana conventions.

**Tradeoff:** The parser needs special handling for numeric literals followed
by duration suffixes. The `m` suffix could collide with a hypothetical variable
named `m`, but since duration suffixes only appear after numeric literals and
identifiers never start with digits, there is no actual ambiguity.

### 4.5 Implicit Type Coercion in Expressions

**Decision:** The expression evaluator coerces types as needed: booleans are
0/1 in arithmetic, numbers become strings in `~` comparisons, etc.

**Rationale:** The existing query parser already does this (comparing a float
field with an integer literal like `cpu > 50`). In a config DSL, forcing
explicit casts (`float(50)`) adds noise without meaningful safety benefit.

**Tradeoff:** Type errors become runtime errors (e.g., `"hello" + 5`). Since
config files are validated at startup before any data is displayed, this is
acceptable: users see clear error messages immediately.

### 4.6 `weight` as Positional Modifier, Not Attribute

**Decision:** `row weight 3 { ... }` rather than `row { weight = 3; ... }`.

**Rationale:** Weight is the primary (and often only) property of rows and
columns. Making it positional reduces nesting and noise. The word `weight`
acts as a modifier keyword, similar to how HCL uses labels after block types.

**Tradeoff:** If rows ever need multiple attributes (e.g., `min_height`), the
syntax would need either a second positional slot (messy) or a fallback to
attribute syntax inside the block. The solution is to keep `weight` positional
and allow additional attributes inside the block:
```
row weight 3 {
    min_height = 5
    col weight 1 { cpu }
}
```

### 4.7 Widget References vs. Inline Widget Blocks

**Decision:** Built-in widgets can be referenced by bare identifier (`cpu`,
`procs`) or with parenthesized options (`cpu(sparkline = true)`). Custom
widgets use the full `widget "name" { ... }` block syntax.

**Rationale:** The common case (using a built-in widget with defaults) should
be as terse as possible. A single word `cpu` inside a `col` block is maximally
readable. When customization is needed, the parenthesized form keeps it inline
without requiring a separate block. The full `widget` block is reserved for
custom/composite widgets that need multiple attributes.

**Tradeoff:** Two syntactic forms for "put a widget here." The parser
distinguishes them by context: bare identifiers and call-syntax inside `col`
blocks are widget references; `widget "name" { }` is an inline widget definition.

### 4.8 Filter Expressions as a Superset of the Query DSL

**Decision:** The `filter` attribute in `procs()` and the `when` condition in
alerts use the same expression grammar, which is a strict superset of the
existing process query syntax.

**Rationale:** The existing query syntax (`cpu > 5 and name ~ "chrome"`) is
already in use and tested. Users who know the interactive query bar can use
the same expressions in config files without learning a new syntax. The config
DSL extends it with arithmetic, function calls, and dotted paths, but every
valid query-bar expression is also a valid config expression.

**Tradeoff:** The existing query parser is simpler (no arithmetic, no dotted
paths, no functions). The config parser must be a full expression parser that
happens to accept the simple form as a subset. This means two parsers coexist:
the fast runtime query parser (for the interactive query bar) and the full
config expression parser (for startup config evaluation). They share operator
semantics but not implementation.

### 4.9 String Interpolation with `{expr}` in Messages

**Decision:** Alert messages and title strings support `{expr}` interpolation:
`"CPU at {cpu.total}% for 30s"`.

**Rationale:** Alerts need dynamic content. Building messages from
concatenation (`"CPU at " + string(cpu.total) + "%"`) is noisy. The `{}`
interpolation syntax is familiar from Python f-strings and Rust format macros.
Single braces are used (not `${}`) to minimize punctuation.

**Tradeoff:** Literal `{` in strings requires escaping (`{{`). This is
extremely rare in monitoring messages and not a real concern.

### 4.10 `~` as the Contains/Match Operator

**Decision:** Keep `~` from the existing query DSL for substring matching.

**Rationale:** Already established in the codebase (`name ~ "chrome"` means
"name contains chrome"). Tilde for pattern matching has precedent in
PostgreSQL (`~` for regex) and Ruby (`=~`). In this DSL it performs
case-insensitive substring matching, which is the common case for process
filtering.

**Tradeoff:** Users may expect `~` to mean regex matching. The DSL could later
add `~=` for regex if needed, keeping `~` for the simpler substring check.

### 4.11 No Semicolons or Commas Between Attributes

**Decision:** Attributes within a block are separated by newlines. No
semicolons are required. Commas are only used inside list literals (`[a, b]`)
and function argument lists (`max(a, b)`).

**Rationale:** Reduces punctuation noise. HCL and Nix both demonstrate that
newline-separated attributes are readable without terminators. The parser
treats newlines as statement separators within blocks.

**Tradeoff:** One-liner blocks become ambiguous without semicolons. The solution
is to allow optional semicolons as statement separators for users who want
compact syntax: `global { refresh = 1s; title = "box" }`. The canonical
multi-line style uses newlines.

### 4.12 Naming Convention: Lowercase with Underscores

**Decision:** All keywords, identifiers, and built-in names use `snake_case`:
`ram_percent`, `send_rate`, `show_per_core`, `json_path`.

**Rationale:** Consistent with the existing Go struct field names when
converted to their JSON tags (which are already snake_case in the
`metrics.Snapshot` struct). Underscores are more readable than camelCase in a
config file context and match the convention used by HCL, TOML, and Prometheus.

**Tradeoff:** Go code uses camelCase internally. The config parser must map
snake_case attribute names to Go struct fields. This mapping already exists
in the JSON tags.

### 4.13 Mount Point Aliasing in Disk Paths

**Decision:** Mount points are aliased to safe identifiers: `/` becomes `root`,
`/data` becomes `data`, `/var/log` becomes `var_log`. Users write
`disk.root.percent`, not `disk."/".percent`.

**Rationale:** Dotted paths with string keys (`disk."/data"`) look awkward and
require the parser to handle quoted segments. Simple alphanumeric aliases are
cleaner. The alias is the mount point with leading `/` stripped and remaining
`/` replaced by `_`.

**Tradeoff:** Users must know the alias convention. Mitigated by clear error
messages: `unknown disk path "disk./data" -- did you mean "disk.data"?`

### 4.14 `for` as Temporal Qualifier in Alerts

**Decision:** `for = 30s` means "the condition must be true continuously for
30 seconds before the alert fires."

**Rationale:** Directly borrowed from Prometheus alerting rules, where `for`
is well-understood. Prevents flapping: a single CPU spike does not trigger
a sustained-load alert.

**Tradeoff:** `for` is a common keyword in loop constructs. Since TermDash
configs have no loops, there is no conflict. The keyword reads naturally:
"when cpu.total > 90 for 30s."

### 4.15 Exec Actions in Alerts -- Security Considerations

**Decision:** The `exec` attribute in alert actions runs a shell command.
This is intentional and explicitly opt-in (users must write the command in
their own config file).

**Rationale:** Server monitoring configs often need to send webhooks, write
to syslog, or trigger external tools. Shell exec is the simplest integration
point that requires no plugin system.

**Tradeoff:** Arbitrary command execution is a security surface. Mitigated by:
config files require explicit filesystem access, termdash runs as the
invoking user (no privilege escalation), and exec commands inherit the user's
environment. A future `--no-exec` flag could disable exec actions entirely
for untrusted configs.

### 4.16 No Import or Include System (Yet)

**Decision:** Version 0.1 has no `import` or `include` statement. One file,
one config.

**Rationale:** Imports add significant complexity (circular dependency detection,
path resolution, override semantics). For a terminal monitor, a single config
file is almost always sufficient. The entire server config example above is
under 200 lines.

**Tradeoff:** Users who want to share theme definitions or alert rule libraries
must copy-paste. A future `include "path"` directive could be added as a
simple textual inclusion (like C `#include`) without affecting the rest of
the grammar.

### 4.17 Widget Options: Parenthesized vs. Block Syntax

**Decision:** Built-in widgets support two configuration styles:
- Inline: `cpu(sparkline = true, show_per_core = true)`
- Block (for complex configs): uses the full `widget "name" { }` form

**Rationale:** Most widget customization involves 1-3 boolean flags. The
parenthesized form keeps the layout section compact and scannable. The full
block form is available when a widget needs many options, conditional styling,
or custom data rows.

**Tradeoff:** The parenthesized form uses `key = value` pairs separated by
commas inside `()`, which looks like a function call but is not one. This could
confuse users who expect positional arguments. Clarity is maintained by always
requiring named keys.

### 4.18 Expression Evaluation Context

**Decision:** Expressions in different contexts have different available
variables:
- **Metric `expr`:** System-level metrics + other defined metrics + sources
- **Alert `when`:** System-level metrics + defined metrics + sources
- **Process `filter`:** Per-process fields (name, cpu, mem, etc.)
- **Style `when`:** Depends on style target (system-level for panels, per-process for rows)
- **`procs.any(expr)`:** Per-process fields within the predicate

**Rationale:** This matches natural expectations. When you write a filter for
the process table, `cpu` means "this process's CPU." When you write an alert
condition, `cpu.total` means "total system CPU."

**Tradeoff:** The same word `cpu` means different things in different contexts.
This is intentional: in a process filter context, `cpu` is unambiguous (it is
the process's CPU field, matching the existing query DSL). In a system context,
`cpu` alone would be ambiguous, so the dotted form `cpu.total` is required. The
parser enforces this: bare `cpu` in a system expression context is an error with
the message `did you mean "cpu.total"?`
