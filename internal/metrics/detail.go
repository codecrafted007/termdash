package metrics

import (
	"fmt"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessDetail holds comprehensive information about a single process.
type ProcessDetail struct {
	PID        int32
	Name       string
	Cmdline    string
	Exe        string
	User       string
	CreateTime time.Time
	CPUPercent float64
	MemPercent float32
	MemRSS     uint64
	MemVMS     uint64
	NumThreads int32
	NumFDs     int32
	ConnCount  int
	Environ    []string
	Status     string
	Nice       int32
	ParentPID  int32
	Err        string // accumulated per-field errors
}

// CollectProcessDetail gathers detailed information for a single process.
// Only called on-demand when entering the detail view.
func CollectProcessDetail(pid int32, connCount int) (*ProcessDetail, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("process %d not found: %w", pid, err)
	}

	d := &ProcessDetail{
		PID:       pid,
		ConnCount: connCount,
	}

	var errs []string

	if name, err := p.Name(); err == nil {
		d.Name = name
	} else {
		errs = append(errs, fmt.Sprintf("name: %v", err))
	}

	if cmdline, err := p.Cmdline(); err == nil {
		d.Cmdline = cmdline
	} else {
		errs = append(errs, fmt.Sprintf("cmdline: %v", err))
	}

	if exe, err := p.Exe(); err == nil {
		d.Exe = exe
	} else {
		errs = append(errs, fmt.Sprintf("exe: %v", err))
	}

	if user, err := p.Username(); err == nil {
		d.User = user
	} else {
		errs = append(errs, fmt.Sprintf("user: %v", err))
	}

	if ct, err := p.CreateTime(); err == nil {
		d.CreateTime = time.UnixMilli(ct)
	} else {
		errs = append(errs, fmt.Sprintf("createtime: %v", err))
	}

	if cpu, err := p.CPUPercent(); err == nil {
		d.CPUPercent = cpu
	} else {
		errs = append(errs, fmt.Sprintf("cpu: %v", err))
	}

	if mem, err := p.MemoryPercent(); err == nil {
		d.MemPercent = mem
	} else {
		errs = append(errs, fmt.Sprintf("mem: %v", err))
	}

	if mi, err := p.MemoryInfo(); err == nil && mi != nil {
		d.MemRSS = mi.RSS
		d.MemVMS = mi.VMS
	} else if err != nil {
		errs = append(errs, fmt.Sprintf("meminfo: %v", err))
	}

	if threads, err := p.NumThreads(); err == nil {
		d.NumThreads = threads
	} else {
		errs = append(errs, fmt.Sprintf("threads: %v", err))
	}

	if fds, err := p.NumFDs(); err == nil {
		d.NumFDs = fds
	} else {
		d.NumFDs = -1 // N/A (common on macOS without root)
		errs = append(errs, fmt.Sprintf("fds: %v", err))
	}

	if env, err := p.Environ(); err == nil {
		// Truncate each var and limit count
		maxVars := 50
		if len(env) > maxVars {
			env = env[:maxVars]
		}
		for i, v := range env {
			if len(v) > 200 {
				env[i] = v[:200] + "..."
			}
		}
		d.Environ = env
	} else {
		errs = append(errs, fmt.Sprintf("environ: %v", err))
	}

	if statuses, err := p.Status(); err == nil && len(statuses) > 0 {
		d.Status = statuses[0]
	} else if err != nil {
		d.Status = "?"
		errs = append(errs, fmt.Sprintf("status: %v", err))
	}

	if nice, err := p.Nice(); err == nil {
		d.Nice = nice
	} else {
		errs = append(errs, fmt.Sprintf("nice: %v", err))
	}

	if ppid, err := p.Ppid(); err == nil {
		d.ParentPID = ppid
	} else {
		errs = append(errs, fmt.Sprintf("ppid: %v", err))
	}

	if len(errs) > 0 {
		d.Err = strings.Join(errs, "; ")
	}

	return d, nil
}
