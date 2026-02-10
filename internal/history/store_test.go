package history

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/brajesh/termdash/internal/metrics"
)

func testSnapshot(ts time.Time) metrics.Snapshot {
	return metrics.Snapshot{
		Timestamp: ts,
		CPU: metrics.CPUMetrics{
			Total:   45.5,
			PerCore: []float64{30.0, 60.0, 40.0, 50.0},
		},
		Memory: metrics.MemoryMetrics{
			TotalRAM:    16000000000,
			UsedRAM:     8000000000,
			RAMPercent:  50.0,
			TotalSwap:   4000000000,
			UsedSwap:    1000000000,
			SwapPercent: 25.0,
		},
		Disks: []metrics.DiskMetrics{
			{MountPoint: "/", Device: "/dev/sda1", Total: 500000000000, Used: 250000000000, Percent: 50.0},
			{MountPoint: "/home", Device: "/dev/sda2", Total: 1000000000000, Used: 300000000000, Percent: 30.0},
		},
		Network: metrics.NetworkMetrics{
			BytesSent: 1000000,
			BytesRecv: 2000000,
			SendRate:  5000.0,
			RecvRate:  10000.0,
		},
		Processes: []metrics.ProcessInfo{
			{PID: 1, Name: "init", User: "root", CPUPercent: 0.1, MemPercent: 0.5, MemRSS: 1024},
			{PID: 100, Name: "chrome", User: "alice", CPUPercent: 25.0, MemPercent: 10.0, MemRSS: 500000000},
			{PID: 200, Name: "code", User: "alice", CPUPercent: 15.0, MemPercent: 8.0, MemRSS: 300000000},
			{PID: 300, Name: "slack", User: "alice", CPUPercent: 5.0, MemPercent: 4.0, MemRSS: 200000000},
			{PID: 400, Name: "docker", User: "root", CPUPercent: 3.0, MemPercent: 2.0, MemRSS: 100000000},
		},
		Hostname:      "testhost",
		OS:            "linux",
		UptimeSeconds: 3600,
		Uptime:        time.Hour,
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestWriteAndLoadSnapshot(t *testing.T) {
	store := openTestStore(t)
	ts := time.Now()
	snap := testSnapshot(ts)
	connCounts := map[int32]int{100: 5, 200: 3}

	if err := store.WriteSnapshot(snap, connCounts, 0); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	loaded, loadedConns, err := store.LoadSnapshot(ts.UnixMilli())
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadSnapshot returned nil")
	}

	// Verify system metrics.
	if loaded.CPU.Total != snap.CPU.Total {
		t.Errorf("CPU.Total: got %v, want %v", loaded.CPU.Total, snap.CPU.Total)
	}
	if len(loaded.CPU.PerCore) != len(snap.CPU.PerCore) {
		t.Errorf("CPU.PerCore length: got %d, want %d", len(loaded.CPU.PerCore), len(snap.CPU.PerCore))
	}
	if loaded.Memory.TotalRAM != snap.Memory.TotalRAM {
		t.Errorf("Memory.TotalRAM: got %d, want %d", loaded.Memory.TotalRAM, snap.Memory.TotalRAM)
	}
	if loaded.Hostname != snap.Hostname {
		t.Errorf("Hostname: got %q, want %q", loaded.Hostname, snap.Hostname)
	}

	// Verify disks.
	if len(loaded.Disks) != len(snap.Disks) {
		t.Errorf("Disks count: got %d, want %d", len(loaded.Disks), len(snap.Disks))
	}

	// Verify processes.
	if len(loaded.Processes) != len(snap.Processes) {
		t.Errorf("Processes count: got %d, want %d", len(loaded.Processes), len(snap.Processes))
	}

	// Verify connection counts.
	if loadedConns[100] != 5 {
		t.Errorf("connCounts[100]: got %d, want 5", loadedConns[100])
	}
	if loadedConns[200] != 3 {
		t.Errorf("connCounts[200]: got %d, want 3", loadedConns[200])
	}
}

func TestMaxProcsLimitsProcesses(t *testing.T) {
	store := openTestStore(t)
	snap := testSnapshot(time.Now())
	// snap has 5 processes; limit to 2
	if err := store.WriteSnapshot(snap, nil, 2); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	loaded, _, err := store.LoadSnapshot(snap.Timestamp.UnixMilli())
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if len(loaded.Processes) != 2 {
		t.Errorf("Processes count: got %d, want 2", len(loaded.Processes))
	}
	// Top 2 by CPU should be chrome (25%) and code (15%).
	for _, p := range loaded.Processes {
		if p.Name != "chrome" && p.Name != "code" {
			t.Errorf("unexpected process %q in top 2", p.Name)
		}
	}
}

func TestMaxProcsZeroStoresAll(t *testing.T) {
	store := openTestStore(t)
	snap := testSnapshot(time.Now())
	if err := store.WriteSnapshot(snap, nil, 0); err != nil {
		t.Fatalf("WriteSnapshot: %v", err)
	}

	loaded, _, err := store.LoadSnapshot(snap.Timestamp.UnixMilli())
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if len(loaded.Processes) != len(snap.Processes) {
		t.Errorf("Processes count: got %d, want %d", len(loaded.Processes), len(snap.Processes))
	}
}

func TestListTimestamps(t *testing.T) {
	store := openTestStore(t)
	base := time.Now()

	for i := 0; i < 5; i++ {
		snap := testSnapshot(base.Add(time.Duration(i) * 10 * time.Second))
		if err := store.WriteSnapshot(snap, nil, 0); err != nil {
			t.Fatalf("WriteSnapshot[%d]: %v", i, err)
		}
	}

	// Query all.
	ts, err := store.ListTimestamps(base.UnixMilli(), base.Add(time.Minute).UnixMilli())
	if err != nil {
		t.Fatalf("ListTimestamps: %v", err)
	}
	if len(ts) != 5 {
		t.Errorf("ListTimestamps count: got %d, want 5", len(ts))
	}

	// Query subset (timestamps 1-3, 10s to 30s).
	ts, err = store.ListTimestamps(
		base.Add(10*time.Second).UnixMilli(),
		base.Add(30*time.Second).UnixMilli(),
	)
	if err != nil {
		t.Fatalf("ListTimestamps subset: %v", err)
	}
	if len(ts) != 3 {
		t.Errorf("ListTimestamps subset count: got %d, want 3", len(ts))
	}
}

func TestCleanup(t *testing.T) {
	store := openTestStore(t)
	now := time.Now()

	// Write old snapshot (2 hours ago).
	old := testSnapshot(now.Add(-2 * time.Hour))
	if err := store.WriteSnapshot(old, nil, 0); err != nil {
		t.Fatalf("WriteSnapshot old: %v", err)
	}

	// Write recent snapshot.
	recent := testSnapshot(now)
	if err := store.WriteSnapshot(recent, nil, 0); err != nil {
		t.Fatalf("WriteSnapshot recent: %v", err)
	}

	// Cleanup with 1h retention should remove the old one.
	removed, err := store.Cleanup(time.Hour)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if removed != 1 {
		t.Errorf("Cleanup removed: got %d, want 1", removed)
	}

	// Only recent should remain.
	ts, err := store.ListTimestamps(0, now.Add(time.Minute).UnixMilli())
	if err != nil {
		t.Fatalf("ListTimestamps after cleanup: %v", err)
	}
	if len(ts) != 1 {
		t.Errorf("Remaining snapshots: got %d, want 1", len(ts))
	}
}

func TestLoadSnapshotFindsNearest(t *testing.T) {
	store := openTestStore(t)
	base := time.Now()

	t1 := base
	t2 := base.Add(10 * time.Second)
	t3 := base.Add(20 * time.Second)

	for _, ts := range []time.Time{t1, t2, t3} {
		snap := testSnapshot(ts)
		if err := store.WriteSnapshot(snap, nil, 0); err != nil {
			t.Fatalf("WriteSnapshot: %v", err)
		}
	}

	// Query a time between t1 and t2 but closer to t2.
	queryTS := base.Add(8 * time.Second).UnixMilli()
	loaded, _, err := store.LoadSnapshot(queryTS)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadSnapshot returned nil")
	}
	if loaded.Timestamp.UnixMilli() != t2.UnixMilli() {
		t.Errorf("LoadSnapshot nearest: got %d, want %d", loaded.Timestamp.UnixMilli(), t2.UnixMilli())
	}
}

func TestLoadSystemSeries(t *testing.T) {
	store := openTestStore(t)
	base := time.Now()

	for i := 0; i < 3; i++ {
		snap := testSnapshot(base.Add(time.Duration(i) * 10 * time.Second))
		snap.CPU.Total = float64(10 * (i + 1)) // 10, 20, 30
		if err := store.WriteSnapshot(snap, nil, 0); err != nil {
			t.Fatalf("WriteSnapshot[%d]: %v", i, err)
		}
	}

	series, err := store.LoadSystemSeries(base.UnixMilli(), base.Add(time.Minute).UnixMilli())
	if err != nil {
		t.Fatalf("LoadSystemSeries: %v", err)
	}
	if len(series) != 3 {
		t.Fatalf("LoadSystemSeries count: got %d, want 3", len(series))
	}

	// Verify system data is present.
	if series[0].CPU.Total != 10.0 {
		t.Errorf("series[0].CPU.Total: got %v, want 10", series[0].CPU.Total)
	}
	if series[2].CPU.Total != 30.0 {
		t.Errorf("series[2].CPU.Total: got %v, want 30", series[2].CPU.Total)
	}

	// Verify no processes in system series (they're not loaded).
	for i, s := range series {
		if len(s.Processes) != 0 {
			t.Errorf("series[%d] should have no processes, got %d", i, len(s.Processes))
		}
	}
}

func TestLoadSnapshotEmptyDB(t *testing.T) {
	store := openTestStore(t)
	loaded, conns, err := store.LoadSnapshot(time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("LoadSnapshot on empty DB: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil snapshot from empty DB")
	}
	if conns != nil {
		t.Error("expected nil connCounts from empty DB")
	}
}

func TestDBCreatedAutomatically(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sub", "dir", "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open nested path: %v", err)
	}
	store.Close()
}

func TestFormatReplayOffset(t *testing.T) {
	now := time.Now().UnixMilli()
	tests := []struct {
		replayTS int64
		want     string
	}{
		{now - 200_000, "-3m 20s"},
		{now - 30_000, "-30s"},
		{now - 0, "-0s"},
	}
	for _, tt := range tests {
		got := FormatReplayOffset(tt.replayTS, now)
		if got != tt.want {
			t.Errorf("FormatReplayOffset(%d, %d) = %q, want %q", tt.replayTS, now, got, tt.want)
		}
	}
}
