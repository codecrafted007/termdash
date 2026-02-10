package metrics

import (
	"sync"
	"time"
)

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

// SystemSnapshot is a point-in-time collection of system metrics (excluding processes).
type SystemSnapshot struct {
	Timestamp     time.Time      `json:"timestamp"`
	CPU           CPUMetrics     `json:"cpu"`
	Memory        MemoryMetrics  `json:"memory"`
	Disks         []DiskMetrics  `json:"disks"`
	Network       NetworkMetrics `json:"network"`
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
	procCollector *ProcessCollector
}

// NewCollector returns an initialized Collector.
func NewCollector() *Collector {
	return &Collector{
		procCollector: NewProcessCollector(),
	}
}

// CollectSystem gathers system-level metrics (CPU, memory, disks, host info)
// in parallel via goroutines, with network collected on the calling goroutine
// (stateful: tracks previous byte counts for rate calculation).
func (c *Collector) CollectSystem() (SystemSnapshot, error) {
	now := time.Now()

	var (
		cpuMetrics  CPUMetrics
		memMetrics  MemoryMetrics
		diskMetrics []DiskMetrics
		hostname    string
		osInfo      string
		uptime      time.Duration
		cpuErr      error
		memErr      error
		diskErr     error
	)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		cpuMetrics, cpuErr = collectCPU()
	}()

	go func() {
		defer wg.Done()
		memMetrics, memErr = collectMemory()
	}()

	go func() {
		defer wg.Done()
		diskMetrics, diskErr = collectDisks()
		hostname, osInfo, uptime = collectHostInfo()
	}()

	// Network stays on the calling goroutine (stateful: reads/writes prevBytesSent etc.)
	netMetrics, netErr := c.collectNetwork(now)

	wg.Wait()

	// Return first error encountered
	for _, err := range []error{cpuErr, memErr, diskErr, netErr} {
		if err != nil {
			return SystemSnapshot{}, err
		}
	}

	return SystemSnapshot{
		Timestamp:     now,
		CPU:           cpuMetrics,
		Memory:        memMetrics,
		Disks:         diskMetrics,
		Network:       netMetrics,
		Hostname:      hostname,
		OS:            osInfo,
		Uptime:        uptime,
		UptimeSeconds: int64(uptime.Seconds()),
	}, nil
}

// CollectProcesses gathers process metrics. Safe for concurrent use
// (ProcessCollector is internally synchronized with a mutex).
func (c *Collector) CollectProcesses() ([]ProcessInfo, error) {
	return c.procCollector.CollectAll()
}

// Collect gathers a full system metrics Snapshot.
// Kept for backward compatibility (tests, export).
func (c *Collector) Collect() (Snapshot, error) {
	sys, err := c.CollectSystem()
	if err != nil {
		return Snapshot{}, err
	}

	procs, _ := c.CollectProcesses()

	return Snapshot{
		Timestamp:     sys.Timestamp,
		CPU:           sys.CPU,
		Memory:        sys.Memory,
		Disks:         sys.Disks,
		Network:       sys.Network,
		Processes:     procs,
		Hostname:      sys.Hostname,
		OS:            sys.OS,
		Uptime:        sys.Uptime,
		UptimeSeconds: sys.UptimeSeconds,
	}, nil
}
