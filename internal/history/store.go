package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/brajesh/termdash/internal/metrics"
	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS snapshots (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    ts              INTEGER NOT NULL,
    cpu_total       REAL,
    cpu_per_core    TEXT,
    mem_total       INTEGER,
    mem_used        INTEGER,
    mem_percent     REAL,
    swap_total      INTEGER,
    swap_used       INTEGER,
    swap_percent    REAL,
    net_bytes_sent  INTEGER,
    net_bytes_recv  INTEGER,
    net_send_rate   REAL,
    net_recv_rate   REAL,
    hostname        TEXT,
    os_name         TEXT,
    uptime_seconds  INTEGER
);
CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(ts);

CREATE TABLE IF NOT EXISTS disks (
    snapshot_id  INTEGER NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
    mount_point  TEXT,
    device       TEXT,
    total        INTEGER,
    used         INTEGER,
    percent      REAL
);
CREATE INDEX IF NOT EXISTS idx_disks_sid ON disks(snapshot_id);

CREATE TABLE IF NOT EXISTS processes (
    snapshot_id  INTEGER NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
    pid          INTEGER,
    name         TEXT,
    username     TEXT,
    cpu_percent  REAL,
    mem_percent  REAL,
    mem_rss      INTEGER,
    conn_count   INTEGER
);
CREATE INDEX IF NOT EXISTS idx_procs_sid ON processes(snapshot_id);
`

// Store provides SQLite-backed persistence for system snapshots.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at dbPath and initializes the schema.
func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("history: create directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("history: open db: %w", err)
	}

	// Set pragmas for performance and correctness.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("history: %s: %w", pragma, err)
		}
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("history: create tables: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// WriteSnapshot persists a snapshot to the database. If maxProcs > 0,
// only the top N processes (by CPU) are stored.
func (s *Store) WriteSnapshot(snap metrics.Snapshot, connCounts map[int32]int, maxProcs int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("history: begin tx: %w", err)
	}
	defer tx.Rollback()

	cpuJSON, _ := json.Marshal(snap.CPU.PerCore)

	res, err := tx.Exec(`INSERT INTO snapshots
		(ts, cpu_total, cpu_per_core, mem_total, mem_used, mem_percent,
		 swap_total, swap_used, swap_percent,
		 net_bytes_sent, net_bytes_recv, net_send_rate, net_recv_rate,
		 hostname, os_name, uptime_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.Timestamp.UnixMilli(),
		snap.CPU.Total,
		string(cpuJSON),
		int64(snap.Memory.TotalRAM),
		int64(snap.Memory.UsedRAM),
		snap.Memory.RAMPercent,
		int64(snap.Memory.TotalSwap),
		int64(snap.Memory.UsedSwap),
		snap.Memory.SwapPercent,
		int64(snap.Network.BytesSent),
		int64(snap.Network.BytesRecv),
		snap.Network.SendRate,
		snap.Network.RecvRate,
		snap.Hostname,
		snap.OS,
		snap.UptimeSeconds,
	)
	if err != nil {
		return fmt.Errorf("history: insert snapshot: %w", err)
	}

	snapID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("history: last insert id: %w", err)
	}

	// Insert disk metrics.
	for _, d := range snap.Disks {
		if _, err := tx.Exec(`INSERT INTO disks (snapshot_id, mount_point, device, total, used, percent)
			VALUES (?, ?, ?, ?, ?, ?)`,
			snapID, d.MountPoint, d.Device, int64(d.Total), int64(d.Used), d.Percent,
		); err != nil {
			return fmt.Errorf("history: insert disk: %w", err)
		}
	}

	// Sort processes by CPU descending for top-N selection.
	procs := make([]metrics.ProcessInfo, len(snap.Processes))
	copy(procs, snap.Processes)
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPUPercent > procs[j].CPUPercent
	})

	limit := len(procs)
	if maxProcs > 0 && maxProcs < limit {
		limit = maxProcs
	}

	for _, p := range procs[:limit] {
		cc := connCounts[p.PID]
		if _, err := tx.Exec(`INSERT INTO processes
			(snapshot_id, pid, name, username, cpu_percent, mem_percent, mem_rss, conn_count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			snapID, p.PID, p.Name, p.User, p.CPUPercent, float64(p.MemPercent), int64(p.MemRSS), cc,
		); err != nil {
			return fmt.Errorf("history: insert process: %w", err)
		}
	}

	return tx.Commit()
}

// LoadSnapshot loads the snapshot closest to the given timestamp (unix millis).
func (s *Store) LoadSnapshot(ts int64) (*metrics.Snapshot, map[int32]int, error) {
	// Find the nearest snapshot by absolute time distance.
	row := s.db.QueryRow(`SELECT id, ts, cpu_total, cpu_per_core,
		mem_total, mem_used, mem_percent,
		swap_total, swap_used, swap_percent,
		net_bytes_sent, net_bytes_recv, net_send_rate, net_recv_rate,
		hostname, os_name, uptime_seconds
		FROM snapshots ORDER BY ABS(ts - ?) LIMIT 1`, ts)

	var (
		snapID       int64
		tsMillis     int64
		cpuTotal     float64
		cpuPerCore   string
		memTotal     int64
		memUsed      int64
		memPct       float64
		swapTotal    int64
		swapUsed     int64
		swapPct      float64
		netSent      int64
		netRecv      int64
		netSendRate  float64
		netRecvRate  float64
		hostname     string
		osName       string
		uptimeSec    int64
	)

	err := row.Scan(&snapID, &tsMillis, &cpuTotal, &cpuPerCore,
		&memTotal, &memUsed, &memPct,
		&swapTotal, &swapUsed, &swapPct,
		&netSent, &netRecv, &netSendRate, &netRecvRate,
		&hostname, &osName, &uptimeSec)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("history: load snapshot: %w", err)
	}

	var perCore []float64
	json.Unmarshal([]byte(cpuPerCore), &perCore)

	snap := &metrics.Snapshot{
		Timestamp: time.UnixMilli(tsMillis),
		CPU: metrics.CPUMetrics{
			Total:   cpuTotal,
			PerCore: perCore,
		},
		Memory: metrics.MemoryMetrics{
			TotalRAM:    uint64(memTotal),
			UsedRAM:     uint64(memUsed),
			RAMPercent:  memPct,
			TotalSwap:   uint64(swapTotal),
			UsedSwap:    uint64(swapUsed),
			SwapPercent: swapPct,
		},
		Network: metrics.NetworkMetrics{
			BytesSent: uint64(netSent),
			BytesRecv: uint64(netRecv),
			SendRate:  netSendRate,
			RecvRate:  netRecvRate,
		},
		Hostname:      hostname,
		OS:            osName,
		UptimeSeconds: uptimeSec,
		Uptime:        time.Duration(uptimeSec) * time.Second,
	}

	// Load disks.
	diskRows, err := s.db.Query(`SELECT mount_point, device, total, used, percent
		FROM disks WHERE snapshot_id = ?`, snapID)
	if err != nil {
		return nil, nil, fmt.Errorf("history: load disks: %w", err)
	}
	defer diskRows.Close()

	for diskRows.Next() {
		var d metrics.DiskMetrics
		var total, used int64
		if err := diskRows.Scan(&d.MountPoint, &d.Device, &total, &used, &d.Percent); err != nil {
			return nil, nil, fmt.Errorf("history: scan disk: %w", err)
		}
		d.Total = uint64(total)
		d.Used = uint64(used)
		snap.Disks = append(snap.Disks, d)
	}

	// Load processes.
	procRows, err := s.db.Query(`SELECT pid, name, username, cpu_percent, mem_percent, mem_rss, conn_count
		FROM processes WHERE snapshot_id = ?`, snapID)
	if err != nil {
		return nil, nil, fmt.Errorf("history: load processes: %w", err)
	}
	defer procRows.Close()

	connCounts := make(map[int32]int)
	for procRows.Next() {
		var p metrics.ProcessInfo
		var memPct float64
		var memRSS int64
		var cc int
		if err := procRows.Scan(&p.PID, &p.Name, &p.User, &p.CPUPercent, &memPct, &memRSS, &cc); err != nil {
			return nil, nil, fmt.Errorf("history: scan process: %w", err)
		}
		p.MemPercent = float32(memPct)
		p.MemRSS = uint64(memRSS)
		p.ConnCount = cc
		connCounts[p.PID] = cc
		snap.Processes = append(snap.Processes, p)
	}

	return snap, connCounts, nil
}

// LoadSystemSeries loads system-only metrics (no processes) for the given time range.
// Useful for populating sparklines during replay.
func (s *Store) LoadSystemSeries(from, to int64) ([]metrics.Snapshot, error) {
	rows, err := s.db.Query(`SELECT ts, cpu_total, cpu_per_core,
		mem_total, mem_used, mem_percent,
		swap_total, swap_used, swap_percent,
		net_bytes_sent, net_bytes_recv, net_send_rate, net_recv_rate,
		hostname, os_name, uptime_seconds
		FROM snapshots WHERE ts >= ? AND ts <= ? ORDER BY ts`, from, to)
	if err != nil {
		return nil, fmt.Errorf("history: load system series: %w", err)
	}
	defer rows.Close()

	var series []metrics.Snapshot
	for rows.Next() {
		var (
			tsMillis    int64
			cpuTotal    float64
			cpuPerCore  string
			memTotal    int64
			memUsed     int64
			memPct      float64
			swapTotal   int64
			swapUsed    int64
			swapPct     float64
			netSent     int64
			netRecv     int64
			netSendRate float64
			netRecvRate float64
			hostname    string
			osName      string
			uptimeSec   int64
		)
		if err := rows.Scan(&tsMillis, &cpuTotal, &cpuPerCore,
			&memTotal, &memUsed, &memPct,
			&swapTotal, &swapUsed, &swapPct,
			&netSent, &netRecv, &netSendRate, &netRecvRate,
			&hostname, &osName, &uptimeSec); err != nil {
			return nil, fmt.Errorf("history: scan series row: %w", err)
		}

		var perCore []float64
		json.Unmarshal([]byte(cpuPerCore), &perCore)

		series = append(series, metrics.Snapshot{
			Timestamp: time.UnixMilli(tsMillis),
			CPU: metrics.CPUMetrics{
				Total:   cpuTotal,
				PerCore: perCore,
			},
			Memory: metrics.MemoryMetrics{
				TotalRAM:    uint64(memTotal),
				UsedRAM:     uint64(memUsed),
				RAMPercent:  memPct,
				TotalSwap:   uint64(swapTotal),
				UsedSwap:    uint64(swapUsed),
				SwapPercent: swapPct,
			},
			Network: metrics.NetworkMetrics{
				BytesSent: uint64(netSent),
				BytesRecv: uint64(netRecv),
				SendRate:  netSendRate,
				RecvRate:  netRecvRate,
			},
			Hostname:      hostname,
			OS:            osName,
			UptimeSeconds: uptimeSec,
			Uptime:        time.Duration(uptimeSec) * time.Second,
		})
	}
	return series, nil
}

// ListTimestamps returns all snapshot timestamps (unix millis) in the given range.
func (s *Store) ListTimestamps(from, to int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT ts FROM snapshots WHERE ts >= ? AND ts <= ? ORDER BY ts`, from, to)
	if err != nil {
		return nil, fmt.Errorf("history: list timestamps: %w", err)
	}
	defer rows.Close()

	var timestamps []int64
	for rows.Next() {
		var ts int64
		if err := rows.Scan(&ts); err != nil {
			return nil, fmt.Errorf("history: scan timestamp: %w", err)
		}
		timestamps = append(timestamps, ts)
	}
	return timestamps, nil
}

// Cleanup deletes snapshots older than the retention duration.
// CASCADE foreign keys handle disk and process rows.
func (s *Store) Cleanup(retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention).UnixMilli()
	res, err := s.db.Exec(`DELETE FROM snapshots WHERE ts < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("history: cleanup: %w", err)
	}
	return res.RowsAffected()
}

// FormatReplayOffset formats the time difference between now and a replay timestamp
// as a human-readable string like "-3m 20s".
func FormatReplayOffset(replayTS, nowTS int64) string {
	diff := time.Duration(nowTS-replayTS) * time.Millisecond
	if diff < 0 {
		diff = -diff
	}
	totalSec := int(math.Round(diff.Seconds()))
	min := totalSec / 60
	sec := totalSec % 60
	if min > 0 {
		return fmt.Sprintf("-%dm %ds", min, sec)
	}
	return fmt.Sprintf("-%ds", sec)
}
