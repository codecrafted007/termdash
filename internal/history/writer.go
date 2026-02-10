package history

import (
	"time"

	"github.com/brajesh/termdash/internal/metrics"
	tea "github.com/charmbracelet/bubbletea"
)

// WriteTickMsg signals that it's time to write a snapshot to the history store.
type WriteTickMsg time.Time

// WriteResultMsg carries the result of a snapshot write.
type WriteResultMsg struct{ Err error }

// CleanupResultMsg carries the result of a retention cleanup.
type CleanupResultMsg struct {
	Removed int64
	Err     error
}

// ReplayLoadMsg carries a loaded snapshot for replay mode.
type ReplayLoadMsg struct {
	Snap       *metrics.Snapshot
	ConnCounts map[int32]int
	Err        error
}

// TimestampsLoadMsg carries the loaded timestamp list for replay mode.
type TimestampsLoadMsg struct {
	Timestamps []int64
	Err        error
}

// WriteTick returns a Cmd that fires a WriteTickMsg after the given interval.
func WriteTick(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return WriteTickMsg(t)
	})
}

// WriteCmd returns a Cmd that writes a snapshot to the store.
func WriteCmd(store *Store, snap metrics.Snapshot, connCounts map[int32]int, maxProcs int) tea.Cmd {
	return func() tea.Msg {
		err := store.WriteSnapshot(snap, connCounts, maxProcs)
		return WriteResultMsg{Err: err}
	}
}

// CleanupCmd returns a Cmd that deletes old snapshots beyond the retention period.
func CleanupCmd(store *Store, retention time.Duration) tea.Cmd {
	return func() tea.Msg {
		removed, err := store.Cleanup(retention)
		return CleanupResultMsg{Removed: removed, Err: err}
	}
}

// LoadSnapshotCmd returns a Cmd that loads a snapshot by timestamp.
func LoadSnapshotCmd(store *Store, ts int64) tea.Cmd {
	return func() tea.Msg {
		snap, conns, err := store.LoadSnapshot(ts)
		return ReplayLoadMsg{Snap: snap, ConnCounts: conns, Err: err}
	}
}

// LoadTimestampsCmd returns a Cmd that loads all timestamps in a range.
func LoadTimestampsCmd(store *Store, from, to int64) tea.Cmd {
	return func() tea.Msg {
		ts, err := store.ListTimestamps(from, to)
		return TimestampsLoadMsg{Timestamps: ts, Err: err}
	}
}
