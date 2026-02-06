package metrics

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExportSnapshot is the top-level struct written to JSON.
// It embeds Snapshot with ConnCounts merged into each ProcessInfo.
type ExportSnapshot struct {
	Snapshot
}

// ExportPath returns a filesystem-safe export path in the user's home directory.
// Format: ~/termdash-{prefix}{timestamp}.{ext}
func ExportPath(prefix, ext string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	ts := time.Now().Format("2006-01-02T15-04-05")
	name := fmt.Sprintf("termdash-%s%s.%s", prefix, ts, ext)
	return filepath.Join(home, name)
}

// ExportJSON writes the snapshot with merged connection counts to a JSON file.
// Returns the path written and any error.
func ExportJSON(snap Snapshot, connCounts map[int32]int, path string) (string, error) {
	// Merge connCounts into process list
	exportSnap := ExportSnapshot{Snapshot: snap}
	procs := make([]ProcessInfo, len(snap.Processes))
	copy(procs, snap.Processes)
	for i := range procs {
		if c, ok := connCounts[procs[i].PID]; ok {
			procs[i].ConnCount = c
		}
	}
	exportSnap.Processes = procs

	data, err := json.MarshalIndent(exportSnap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return path, nil
}

// AppendCSV appends process rows to a CSV file. If writeHeader is true,
// the column header row is written first. The file is opened in append mode.
func AppendCSV(snap Snapshot, connCounts map[int32]int, path string, writeHeader bool) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if writeHeader {
		header := []string{"timestamp", "pid", "name", "user", "cpu_pct", "mem_pct", "rss_bytes", "conn_count"}
		if err := w.Write(header); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
	}

	ts := snap.Timestamp.UTC().Format(time.RFC3339)
	for _, p := range snap.Processes {
		connCount := p.ConnCount
		if c, ok := connCounts[p.PID]; ok {
			connCount = c
		}
		row := []string{
			ts,
			fmt.Sprintf("%d", p.PID),
			p.Name,
			p.User,
			fmt.Sprintf("%.1f", p.CPUPercent),
			fmt.Sprintf("%.1f", p.MemPercent),
			fmt.Sprintf("%d", p.MemRSS),
			fmt.Sprintf("%d", connCount),
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}

	return nil
}
