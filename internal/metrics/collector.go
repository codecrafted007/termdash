package metrics

import "time"

// CPUMetrics holds per-core and total CPU usage.
type CPUMetrics struct {
	PerCore []float64 `json:"per_core"` // percentage per core (0–100)
	Total   float64   `json:"total"`    // overall percentage (0–100)
}

// MemoryMetrics holds RAM and swap usage.
type MemoryMetrics struct {
	TotalRAM    uint64  `json:"total_ram"`
	UsedRAM     uint64  `json:"used_ram"`
	RAMPercent  float64 `json:"ram_percent"`
	TotalSwap   uint64  `json:"total_swap"`
	UsedSwap    uint64  `json:"used_swap"`
	SwapPercent float64 `json:"swap_percent"`
}

// DiskMetrics holds usage for a single mount point.
type DiskMetrics struct {
	MountPoint string  `json:"mount_point"`
	Device     string  `json:"device"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Percent    float64 `json:"percent"`
}

// NetworkMetrics holds aggregate network I/O.
type NetworkMetrics struct {
	BytesSent uint64  `json:"bytes_sent"`
	BytesRecv uint64  `json:"bytes_recv"`
	SendRate  float64 `json:"send_rate"` // bytes per second
	RecvRate  float64 `json:"recv_rate"` // bytes per second
}

// ProcessInfo holds information about a single running process.
type ProcessInfo struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu_pct"`
	MemPercent float32 `json:"mem_pct"`
	User       string  `json:"user"`
	MemRSS     uint64  `json:"rss_bytes"`
	ConnCount  int     `json:"conn_count"` // number of network connections (-1 = unavailable)
}

// Snapshot is a point-in-time collection of all system metrics.
type Snapshot struct {
	Timestamp     time.Time      `json:"timestamp"`
	CPU           CPUMetrics     `json:"cpu"`
	Memory        MemoryMetrics  `json:"memory"`
	Disks         []DiskMetrics  `json:"disks"`
	Network       NetworkMetrics `json:"network"`
	Processes     []ProcessInfo  `json:"processes"`
	Hostname      string         `json:"hostname"`
	OS            string         `json:"os"`
	Uptime        time.Duration  `json:"-"`
	UptimeSeconds int64          `json:"uptime_seconds"`
}

// Collector gathers system metrics and tracks previous values for rate calculations.
type Collector struct {
	prevBytesSent uint64
	prevBytesRecv uint64
	prevTime      time.Time
	initialized   bool
}

// NewCollector returns an initialized Collector.
func NewCollector() *Collector {
	return &Collector{}
}

// Collect gathers a full system metrics Snapshot.
func (c *Collector) Collect() (Snapshot, error) {
	now := time.Now()

	cpuMetrics, err := collectCPU()
	if err != nil {
		return Snapshot{}, err
	}

	memMetrics, err := collectMemory()
	if err != nil {
		return Snapshot{}, err
	}

	diskMetrics, err := collectDisks()
	if err != nil {
		return Snapshot{}, err
	}

	netMetrics, err := c.collectNetwork(now)
	if err != nil {
		return Snapshot{}, err
	}

	procs, _ := collectProcesses(25)

	hostname, osInfo, uptime := collectHostInfo()

	return Snapshot{
		Timestamp:     now,
		CPU:           cpuMetrics,
		Memory:        memMetrics,
		Disks:         diskMetrics,
		Network:       netMetrics,
		Processes:     procs,
		Hostname:      hostname,
		OS:            osInfo,
		Uptime:        uptime,
		UptimeSeconds: int64(uptime.Seconds()),
	}, nil
}
