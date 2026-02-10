package ui

import (
	"fmt"
	"sort"
	"time"

	"github.com/brajesh/termdash/internal/history"
	"github.com/brajesh/termdash/internal/metrics"
	"github.com/brajesh/termdash/internal/ui/panels"
	"github.com/brajesh/termdash/pkg/dsl"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	defaultRefreshInterval = 2 * time.Second
	detailTickInterval     = 1 * time.Second
	historyCapacity        = 60
	detailHistoryCap       = 60
)

// ViewState represents which view is active.
type ViewState int

const (
	ViewDashboard ViewState = iota
	ViewProcessDetail
)

// SortColumn represents the column used for sorting the process table.
type SortColumn int

const (
	SortByCPU SortColumn = iota
	SortByMem
	SortByPID
	SortByName
	SortByConns
)

func (s SortColumn) String() string {
	switch s {
	case SortByCPU:
		return "CPU%"
	case SortByMem:
		return "MEM%"
	case SortByPID:
		return "PID"
	case SortByName:
		return "NAME"
	case SortByConns:
		return "CONN"
	default:
		return "CPU%"
	}
}

// History is a fixed-capacity ring buffer for sparkline data.
type History struct {
	data []float64
	cap  int
}

func NewHistory(capacity int) *History {
	return &History{
		data: make([]float64, 0, capacity),
		cap:  capacity,
	}
}

func (h *History) Push(v float64) {
	if len(h.data) >= h.cap {
		h.data = h.data[1:]
	}
	h.data = append(h.data, v)
}

func (h *History) Values() []float64 {
	out := make([]float64, len(h.data))
	copy(out, h.data)
	return out
}

// Messages.
type tickMsg time.Time
type systemMsg metrics.SystemSnapshot
type processMsg []metrics.ProcessInfo
type errMsg error
type connCountMsg map[int32]int
type processDetailMsg struct {
	detail *metrics.ProcessDetail
	err    error
}
type detailTickMsg time.Time

type exportDoneMsg struct {
	path string
	err  error
}
type csvWriteDoneMsg struct{ err error }
type statusClearMsg struct{}

// Model is the root bubbletea model.
type Model struct {
	config    *dsl.Config
	collector *metrics.Collector
	snapshot  metrics.Snapshot
	cpuHistory  *History
	sendHistory *History
	recvHistory *History
	showHelp    bool
	width       int
	height      int
	err         error
	ready       bool

	// Interactive state
	viewState    ViewState
	cursor       int
	scrollOffset int
	sortColumn   SortColumn
	sortAscending bool
	connCounts   map[int32]int

	// Detail view state
	detailPID        int32
	detailInfo       *metrics.ProcessDetail
	detailCPUHistory *History
	detailMemHistory *History

	// Export state
	statusMsg  string // flash message shown in header area
	csvLogging bool   // true while continuous CSV recording
	csvLogPath string // path to active CSV log file

	// Process grouping
	groupedView    bool            // toggle with 'p' key
	expandedGroups map[string]bool // group name → expanded

	// Connection throttling
	connTickCounter int // counts ticks; collect connections every 3rd tick

	// Query/search state
	queryMode    bool            // true when in query mode
	queryType    QueryType       // search vs filter
	queryInput   textinput.Model // bubbles text input component
	queryFilter  *FilterNode     // parsed filter (nil if invalid/empty)
	queryError   string          // parse error message
	filterLocked bool            // true when filter is applied but not editing

	// History / replay state
	historyStore      *history.Store // nil if history disabled
	historyRetention  time.Duration  // parsed from config
	historyInterval   time.Duration  // parsed from config
	historyMaxProcs   int            // max processes per snapshot
	replayMode        bool
	replayIndex       int      // index into replayTimestamps
	replayTimestamps  []int64  // loaded from DB
	replaySnapshot    *metrics.Snapshot
	replayConnCounts  map[int32]int
}

// New creates a new Model with the default config.
func New() Model {
	return NewWithConfig(dsl.DefaultConfig())
}

// NewWithConfig creates a new Model using the given DSL config.
func NewWithConfig(cfg *dsl.Config) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 256

	return Model{
		config:          cfg,
		collector:       metrics.NewCollector(),
		cpuHistory:      NewHistory(historyCapacity),
		sendHistory:     NewHistory(historyCapacity),
		recvHistory:     NewHistory(historyCapacity),
		width:           80,
		height:          24,
		connCounts:      map[int32]int{},
		connTickCounter: 2, // trigger connection collection on first tick
		expandedGroups:  make(map[string]bool),
		queryInput:      ti,
	}
}

// SetHistoryStore configures the model to use the given history store for
// persistence and replay. Call this before running the program.
func (m *Model) SetHistoryStore(store *history.Store, retention, interval time.Duration, maxProcs int) {
	m.historyStore = store
	m.historyRetention = retention
	m.historyInterval = interval
	m.historyMaxProcs = maxProcs
}

// refreshInterval returns the configured refresh interval, falling back to the default.
func (m Model) refreshInterval() time.Duration {
	if m.config != nil && m.config.Global != nil && m.config.Global.Refresh != "" {
		if d, err := dsl.ParseDuration(m.config.Global.Refresh); err == nil {
			return d
		}
	}
	return defaultRefreshInterval
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		collectSystemCmd(m.collector),
		collectProcessesCmd(m.collector),
	}
	if m.historyStore != nil {
		cmds = append(cmds,
			history.WriteTick(m.historyInterval),
			history.CleanupCmd(m.historyStore, m.historyRetention),
		)
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			collectSystemCmd(m.collector),
			collectProcessesCmd(m.collector),
		)

	case systemMsg:
		sys := metrics.SystemSnapshot(msg)
		m.snapshot.Timestamp = sys.Timestamp
		m.snapshot.CPU = sys.CPU
		m.snapshot.Memory = sys.Memory
		m.snapshot.Disks = sys.Disks
		m.snapshot.Network = sys.Network
		m.snapshot.Hostname = sys.Hostname
		m.snapshot.OS = sys.OS
		m.snapshot.Uptime = sys.Uptime
		m.snapshot.UptimeSeconds = sys.UptimeSeconds
		m.cpuHistory.Push(m.snapshot.CPU.Total)
		m.sendHistory.Push(m.snapshot.Network.SendRate)
		m.recvHistory.Push(m.snapshot.Network.RecvRate)
		m.ready = true
		cmds := []tea.Cmd{tickCmd(m.refreshInterval())}
		// Collect connections every 3rd tick (~6 seconds) to reduce syscalls
		m.connTickCounter++
		if m.connTickCounter >= 3 {
			m.connTickCounter = 0
			cmds = append(cmds, collectConnCountsCmd())
		}
		if m.csvLogging && m.csvLogPath != "" {
			cmds = append(cmds, csvWriteCmd(m.snapshot, m.connCounts, m.csvLogPath, false))
		}
		return m, tea.Batch(cmds...)

	case processMsg:
		m.snapshot.Processes = []metrics.ProcessInfo(msg)
		// Clamp cursor if display row count changed
		rowCount := m.displayRowCount()
		if m.cursor >= rowCount && rowCount > 0 {
			m.cursor = rowCount - 1
		}
		// Clean up expanded groups that no longer exist
		m.cleanupExpandedGroups()
		return m, nil

	case connCountMsg:
		m.connCounts = map[int32]int(msg)
		return m, nil

	case processDetailMsg:
		if msg.err != nil {
			// Process may have exited — return to dashboard
			m.viewState = ViewDashboard
			m.detailInfo = nil
			return m, nil
		}
		m.detailInfo = msg.detail
		if m.detailCPUHistory != nil {
			m.detailCPUHistory.Push(msg.detail.CPUPercent)
		}
		if m.detailMemHistory != nil {
			m.detailMemHistory.Push(float64(msg.detail.MemPercent))
		}
		return m, nil

	case detailTickMsg:
		if m.viewState != ViewProcessDetail {
			return m, nil
		}
		connCount := 0
		if c, ok := m.connCounts[m.detailPID]; ok {
			connCount = c
		}
		return m, tea.Batch(
			collectDetailCmd(m.detailPID, connCount),
			detailTickCmd(),
		)

	case exportDoneMsg:
		if msg.err != nil {
			m.statusMsg = "Export error: " + msg.err.Error()
		} else {
			m.statusMsg = "Exported: " + msg.path
		}
		return m, statusClearCmd(3 * time.Second)

	case csvWriteDoneMsg:
		if msg.err != nil {
			m.statusMsg = "CSV error: " + msg.err.Error()
			m.csvLogging = false
			m.csvLogPath = ""
			return m, statusClearCmd(3 * time.Second)
		}
		return m, nil

	case statusClearMsg:
		m.statusMsg = ""
		return m, nil

	case history.WriteTickMsg:
		if m.historyStore != nil && !m.replayMode && m.ready {
			return m, tea.Batch(
				history.WriteCmd(m.historyStore, m.snapshot, m.connCounts, m.historyMaxProcs),
				history.WriteTick(m.historyInterval),
			)
		}
		return m, history.WriteTick(m.historyInterval)

	case history.WriteResultMsg:
		// Log errors silently; don't crash.
		return m, nil

	case history.CleanupResultMsg:
		return m, nil

	case history.TimestampsLoadMsg:
		if msg.Err != nil || len(msg.Timestamps) == 0 {
			m.replayMode = false
			m.statusMsg = "No history available"
			return m, statusClearCmd(3 * time.Second)
		}
		m.replayTimestamps = msg.Timestamps
		m.replayIndex = len(msg.Timestamps) - 1
		return m, history.LoadSnapshotCmd(m.historyStore, msg.Timestamps[m.replayIndex])

	case history.ReplayLoadMsg:
		if msg.Err != nil || msg.Snap == nil {
			return m, nil
		}
		m.replaySnapshot = msg.Snap
		m.replayConnCounts = msg.ConnCounts
		return m, nil

	case errMsg:
		m.err = msg
		return m, tickCmd(m.refreshInterval())
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle query mode input first
	if m.queryMode {
		return m.handleQueryInput(msg)
	}

	// Global keys using key type for special keys
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		// Exit replay mode
		if m.replayMode {
			m.replayMode = false
			m.replaySnapshot = nil
			m.replayConnCounts = nil
			m.replayTimestamps = nil
			return m, nil
		}
		// Handle Esc for detail view
		if m.viewState == ViewProcessDetail {
			m.viewState = ViewDashboard
			m.detailInfo = nil
			m.detailCPUHistory = nil
			m.detailMemHistory = nil
			return m, nil
		}
		// On dashboard with locked filter, clear the filter
		if m.filterLocked {
			m.filterLocked = false
			m.queryFilter = nil
			m.queryInput.SetValue("")
			m.statusMsg = ""
			return m, nil
		}
		return m, nil
	}

	key := msg.String()

	// Global keys
	switch key {
	case "q":
		return m, tea.Quit
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	// Replay mode keys
	if m.replayMode {
		return m.handleReplayKey(key)
	}

	switch m.viewState {
	case ViewDashboard:
		return m.handleDashboardKey(key)
	case ViewProcessDetail:
		return m.handleDetailKey(key)
	}
	return m, nil
}

func (m Model) handleDashboardKey(key string) (tea.Model, tea.Cmd) {
	rowCount := m.displayRowCount()

	switch key {
	case "j", "down":
		if m.cursor < rowCount-1 {
			m.cursor++
		}
		m.ensureCursorVisible()
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		m.ensureCursorVisible()
		return m, nil

	case "g", "home":
		m.cursor = 0
		m.scrollOffset = 0
		return m, nil

	case "G", "end":
		if rowCount > 0 {
			m.cursor = rowCount - 1
		}
		m.ensureCursorVisible()
		return m, nil

	case "enter":
		return m.handleEnterKey()

	case "s":
		// Cycle sort column
		m.sortColumn = (m.sortColumn + 1) % 5
		m.sortAscending = false
		return m, nil

	case "S":
		// Toggle sort direction
		m.sortAscending = !m.sortAscending
		return m, nil

	case "p":
		// Toggle process grouping
		m.groupedView = !m.groupedView
		// Clamp cursor to new row count
		newCount := m.displayRowCount()
		if m.cursor >= newCount && newCount > 0 {
			m.cursor = newCount - 1
		}
		m.scrollOffset = 0
		return m, nil

	case "e":
		return m, exportJSONCmd(m.snapshot, m.connCounts)

	case "E":
		return m.toggleCSVLogging()

	case "/":
		m.queryMode = true
		m.queryType = QuerySearch
		m.queryInput.SetValue("")
		m.queryInput.Focus()
		m.queryError = ""
		m.filterLocked = false
		return m, textinput.Blink

	case "Q":
		m.queryMode = true
		m.queryType = QueryFilter
		m.queryInput.SetValue("")
		m.queryInput.Focus()
		m.queryError = ""
		m.filterLocked = false
		return m, textinput.Blink

	case "t":
		return m.enterReplayMode()
	}

	return m, nil
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.groupedView {
		rows := m.displayRows()
		if m.cursor >= len(rows) {
			return m, nil
		}
		row := rows[m.cursor]
		if row.IsGroup {
			// Toggle expand/collapse
			name := row.Group.Name
			m.expandedGroups[name] = !m.expandedGroups[name]
			return m, nil
		}
		// Open detail for this process
		if row.Process != nil {
			return m.openProcessDetail(row.Process.PID)
		}
		return m, nil
	}

	// Flat view: use filtered processes
	procs := m.filteredProcesses()
	if len(procs) > 0 && m.cursor < len(procs) {
		return m.openProcessDetail(procs[m.cursor].PID)
	}
	return m, nil
}

func (m Model) openProcessDetail(pid int32) (tea.Model, tea.Cmd) {
	m.viewState = ViewProcessDetail
	m.detailPID = pid
	m.detailCPUHistory = NewHistory(detailHistoryCap)
	m.detailMemHistory = NewHistory(detailHistoryCap)
	m.detailInfo = nil
	connCount := 0
	if c, ok := m.connCounts[pid]; ok {
		connCount = c
	}
	return m, tea.Batch(
		collectDetailCmd(pid, connCount),
		detailTickCmd(),
	)
}

func (m Model) handleDetailKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "e":
		return m, exportJSONCmd(m.snapshot, m.connCounts)

	case "E":
		return m.toggleCSVLogging()
	}
	return m, nil
}

func (m Model) enterReplayMode() (tea.Model, tea.Cmd) {
	if m.historyStore == nil {
		m.statusMsg = "History disabled"
		return m, statusClearCmd(3 * time.Second)
	}
	m.replayMode = true
	now := time.Now()
	from := now.Add(-m.historyRetention).UnixMilli()
	to := now.UnixMilli()
	return m, history.LoadTimestampsCmd(m.historyStore, from, to)
}

func (m Model) handleReplayKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "t":
		// Exit replay mode
		m.replayMode = false
		m.replaySnapshot = nil
		m.replayConnCounts = nil
		m.replayTimestamps = nil
		return m, nil

	case "[", "left":
		// Step back one snapshot
		if m.replayIndex > 0 {
			m.replayIndex--
			return m, history.LoadSnapshotCmd(m.historyStore, m.replayTimestamps[m.replayIndex])
		}
		return m, nil

	case "]", "right":
		// Step forward one snapshot
		if m.replayIndex < len(m.replayTimestamps)-1 {
			m.replayIndex++
			return m, history.LoadSnapshotCmd(m.historyStore, m.replayTimestamps[m.replayIndex])
		}
		return m, nil

	case "{":
		// Jump back ~1 minute (6 snapshots at 10s interval)
		m.replayIndex -= 6
		if m.replayIndex < 0 {
			m.replayIndex = 0
		}
		return m, history.LoadSnapshotCmd(m.historyStore, m.replayTimestamps[m.replayIndex])

	case "}":
		// Jump forward ~1 minute
		m.replayIndex += 6
		if m.replayIndex >= len(m.replayTimestamps) {
			m.replayIndex = len(m.replayTimestamps) - 1
		}
		return m, history.LoadSnapshotCmd(m.historyStore, m.replayTimestamps[m.replayIndex])

	case "j", "down":
		rowCount := m.displayRowCount()
		if m.cursor < rowCount-1 {
			m.cursor++
		}
		m.ensureCursorVisible()
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		m.ensureCursorVisible()
		return m, nil
	}

	return m, nil
}

func (m Model) toggleCSVLogging() (tea.Model, tea.Cmd) {
	if m.csvLogging {
		// Stop recording
		m.csvLogging = false
		m.csvLogPath = ""
		m.statusMsg = "Recording stopped"
		return m, statusClearCmd(3 * time.Second)
	}
	// Start recording
	m.csvLogPath = metrics.ExportPath("log-", "csv")
	m.csvLogging = true
	m.statusMsg = "Recording: " + m.csvLogPath
	return m, csvWriteCmd(m.snapshot, m.connCounts, m.csvLogPath, true)
}

func (m Model) handleQueryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		if m.filterLocked {
			// Clear the locked filter
			m.filterLocked = false
			m.queryFilter = nil
			m.queryInput.SetValue("")
			m.statusMsg = ""
		}
		m.queryMode = false
		m.queryInput.Blur()
		m.queryError = ""
		return m, nil

	case tea.KeyEnter:
		if m.queryInput.Value() != "" {
			if m.queryType == QueryFilter && m.queryFilter == nil && m.queryError != "" {
				// Don't lock invalid filter
				return m, nil
			}
			m.filterLocked = true
			m.statusMsg = "Filter: " + m.queryInput.Value()
		}
		m.queryMode = false
		m.queryInput.Blur()
		return m, nil

	case tea.KeyCtrlC:
		// Allow Ctrl+C to quit even in query mode
		return m, tea.Quit
	}

	// Pass to textinput
	var cmd tea.Cmd
	m.queryInput, cmd = m.queryInput.Update(msg)

	// Parse and apply filter
	m.applyQueryFilter()

	return m, cmd
}

// applyQueryFilter parses the query input and updates the filter state.
func (m *Model) applyQueryFilter() {
	if m.queryType == QuerySearch {
		// Simple search doesn't need parsing
		m.queryFilter = nil
		m.queryError = ""
	} else {
		// Parse DSL query
		filter, err := ParseQuery(m.queryInput.Value())
		if err != nil {
			m.queryError = err.Error()
			m.queryFilter = nil
		} else {
			m.queryError = ""
			m.queryFilter = filter
		}
	}

	// Clamp cursor to filtered results
	count := m.displayRowCount()
	if m.cursor >= count && count > 0 {
		m.cursor = count - 1
	}
	if count == 0 {
		m.cursor = 0
	}
	m.scrollOffset = 0
}

// filteredProcesses returns processes filtered by the current query.
func (m Model) filteredProcesses() []metrics.ProcessInfo {
	procs := m.sortedProcesses()

	// No filter active
	if !m.hasActiveFilter() {
		return procs
	}

	conns := m.activeConnCounts()
	var filtered []metrics.ProcessInfo
	for _, p := range procs {
		connCount := conns[p.PID]
		if m.matchesFilter(p, connCount) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// hasActiveFilter returns true if there's an active filter to apply.
func (m Model) hasActiveFilter() bool {
	if m.queryInput.Value() == "" {
		return false
	}
	if m.queryMode || m.filterLocked {
		return true
	}
	return false
}

// matchesFilter returns true if the process matches the current filter.
func (m Model) matchesFilter(p metrics.ProcessInfo, connCount int) bool {
	query := m.queryInput.Value()
	if query == "" {
		return true
	}

	if m.queryType == QuerySearch {
		return SimpleSearch(p, query)
	}

	if m.queryFilter != nil {
		return m.queryFilter.MatchProcess(p, connCount)
	}

	// No valid filter parsed yet, show all
	return true
}

func (m *Model) ensureCursorVisible() {
	visible := m.visibleRows()
	if visible <= 0 {
		visible = 1
	}
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
}

func (m Model) visibleRows() int {
	// header(1) + summary(3) + tableHeader(2) + netTop(2) + help(1) + padding(2)
	overhead := 11
	rows := m.height - overhead
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m Model) sortedProcesses() []metrics.ProcessInfo {
	snap := m.activeSnapshot()
	conns := m.activeConnCounts()
	procs := make([]metrics.ProcessInfo, len(snap.Processes))
	copy(procs, snap.Processes)

	// Apply connection counts
	for i := range procs {
		if c, ok := conns[procs[i].PID]; ok {
			procs[i].ConnCount = c
		}
	}

	less := func(i, j int) bool {
		switch m.sortColumn {
		case SortByCPU:
			return procs[i].CPUPercent > procs[j].CPUPercent
		case SortByMem:
			return procs[i].MemPercent > procs[j].MemPercent
		case SortByPID:
			return procs[i].PID < procs[j].PID
		case SortByName:
			return procs[i].Name < procs[j].Name
		case SortByConns:
			return procs[i].ConnCount > procs[j].ConnCount
		default:
			return procs[i].CPUPercent > procs[j].CPUPercent
		}
	}

	if m.sortAscending {
		sort.Slice(procs, func(i, j int) bool { return !less(i, j) })
	} else {
		sort.Slice(procs, less)
	}

	return procs
}

// displayRowCount returns the number of rows to display based on current view mode.
func (m Model) displayRowCount() int {
	if !m.groupedView {
		return len(m.filteredProcesses())
	}
	return len(m.displayRows())
}

// displayRows builds the display rows for grouped view.
func (m Model) displayRows() []DisplayRow {
	procs := m.filteredProcesses()
	groups := GroupProcesses(procs, m.activeConnCounts())
	return BuildDisplayRows(groups, m.expandedGroups, m.sortColumn, m.sortAscending)
}

// cleanupExpandedGroups removes expanded state for groups that no longer exist.
func (m *Model) cleanupExpandedGroups() {
	if !m.groupedView || len(m.expandedGroups) == 0 {
		return
	}
	// Build set of current group names
	names := make(map[string]bool)
	for _, p := range m.snapshot.Processes {
		names[p.Name] = true
	}
	// Remove expanded state for missing groups
	for name := range m.expandedGroups {
		if !names[name] {
			delete(m.expandedGroups, name)
		}
	}
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Loading system metrics..."
	}

	if m.err != nil {
		return "\n  Error: " + m.err.Error()
	}

	switch m.viewState {
	case ViewProcessDetail:
		return m.renderDetailView()
	default:
		return m.renderDashboardView()
	}
}

func (m Model) activeSnapshot() metrics.Snapshot {
	if m.replayMode && m.replaySnapshot != nil {
		return *m.replaySnapshot
	}
	return m.snapshot
}

func (m Model) activeConnCounts() map[int32]int {
	if m.replayMode && m.replayConnCounts != nil {
		return m.replayConnCounts
	}
	return m.connCounts
}

func (m Model) renderDashboardView() string {
	width := m.width

	snap := m.activeSnapshot()

	var header string
	if m.replayMode {
		header = m.renderReplayHeader(snap)
	} else {
		header = panels.RenderHeader(snap, m.csvLogging, m.statusMsg)
	}

	procs := m.filteredProcesses()
	visible := m.visibleRows()

	// Account for query bar height when active
	if m.queryMode || m.filterLocked {
		visible--
	}

	// Render query bar
	queryBar := ""
	if m.queryMode || m.filterLocked {
		queryBar = panels.RenderQueryBar(
			panels.QueryType(m.queryType),
			m.queryInput,
			m.queryError,
			m.filterLocked,
			width,
		)
	}

	help := panels.RenderHelp(m.showHelp, false)

	// Use config-driven layout when a layout config is present.
	if m.config != nil && m.config.Layout != nil {
		widgets := m.renderWidgets(procs, visible)
		return RenderConfigLayout(m.config.Layout, width, m.height, widgets, header, queryBar, help)
	}

	// Fallback: legacy hardcoded layout.
	summary := panels.RenderSummary(snap, width)

	// Collapse netTop if terminal is too small
	showNetTop := visible >= 5
	if !showNetTop {
		visible += 2
	}

	var processTable string
	if m.groupedView {
		rows := m.displayRows()
		tableRows := make([]panels.TableRow, len(rows))
		for i, r := range rows {
			tableRows[i] = panels.TableRow{
				IsGroup:  r.IsGroup,
				Expanded: r.Expanded,
				Depth:    r.Depth,
			}
			if r.IsGroup && r.Group != nil {
				tableRows[i].PID = fmt.Sprintf("(%d)", len(r.Group.Processes))
				tableRows[i].Name = r.Group.Name
				tableRows[i].User = r.Group.User
				tableRows[i].CPUPercent = r.Group.CPUPercent
				tableRows[i].MemPercent = float64(r.Group.MemPercent)
				tableRows[i].MemRSS = r.Group.MemRSS
				tableRows[i].ConnCount = r.Group.ConnCount
			} else if r.Process != nil {
				tableRows[i].PID = fmt.Sprintf("%d", r.Process.PID)
				tableRows[i].Name = r.Process.Name
				tableRows[i].User = r.Process.User
				tableRows[i].CPUPercent = r.Process.CPUPercent
				tableRows[i].MemPercent = float64(r.Process.MemPercent)
				tableRows[i].MemRSS = r.Process.MemRSS
				tableRows[i].ConnCount = r.Process.ConnCount
			}
		}
		processTable = panels.RenderGroupedProcessTable(
			tableRows,
			m.cursor, m.scrollOffset, visible,
			panels.SortColumn(m.sortColumn), m.sortAscending,
			width,
		)
	} else {
		processTable = panels.RenderProcessTable(
			procs, m.activeConnCounts(),
			m.cursor, m.scrollOffset, visible,
			panels.SortColumn(m.sortColumn), m.sortAscending,
			width,
		)
	}

	netTop := ""
	if showNetTop {
		netTop = panels.RenderNetTop(procs, m.activeConnCounts(), snap, width)
	}

	return RenderDashboardLayout(width, m.height, header, summary, processTable, queryBar, netTop, help)
}

// renderWidgets returns render functions for each widget, called with the actual column width.
func (m Model) renderWidgets(procs []metrics.ProcessInfo, visible int) map[string]WidgetRenderer {
	snap := m.activeSnapshot()
	conns := m.activeConnCounts()
	widgets := make(map[string]WidgetRenderer)

	widgets["summary"] = func(w int) string {
		return panels.RenderSummary(snap, w)
	}

	widgets["cpu"] = func(w int) string {
		return panels.RenderCPU(snap, m.cpuHistory.Values(), w)
	}

	widgets["memory"] = func(w int) string {
		return panels.RenderMemory(snap, w)
	}

	widgets["nettop"] = func(w int) string {
		return panels.RenderNetTop(procs, conns, snap, w)
	}

	if m.groupedView {
		rows := m.displayRows()
		tableRows := make([]panels.TableRow, len(rows))
		for i, r := range rows {
			tableRows[i] = panels.TableRow{
				IsGroup:  r.IsGroup,
				Expanded: r.Expanded,
				Depth:    r.Depth,
			}
			if r.IsGroup && r.Group != nil {
				tableRows[i].PID = fmt.Sprintf("(%d)", len(r.Group.Processes))
				tableRows[i].Name = r.Group.Name
				tableRows[i].User = r.Group.User
				tableRows[i].CPUPercent = r.Group.CPUPercent
				tableRows[i].MemPercent = float64(r.Group.MemPercent)
				tableRows[i].MemRSS = r.Group.MemRSS
				tableRows[i].ConnCount = r.Group.ConnCount
			} else if r.Process != nil {
				tableRows[i].PID = fmt.Sprintf("%d", r.Process.PID)
				tableRows[i].Name = r.Process.Name
				tableRows[i].User = r.Process.User
				tableRows[i].CPUPercent = r.Process.CPUPercent
				tableRows[i].MemPercent = float64(r.Process.MemPercent)
				tableRows[i].MemRSS = r.Process.MemRSS
				tableRows[i].ConnCount = r.Process.ConnCount
			}
		}
		widgets["procs"] = func(w int) string {
			return panels.RenderGroupedProcessTable(
				tableRows,
				m.cursor, m.scrollOffset, visible,
				panels.SortColumn(m.sortColumn), m.sortAscending,
				w,
			)
		}
	} else {
		widgets["procs"] = func(w int) string {
			return panels.RenderProcessTable(
				procs, conns,
				m.cursor, m.scrollOffset, visible,
				panels.SortColumn(m.sortColumn), m.sortAscending,
				w,
			)
		}
	}

	return widgets
}

func (m Model) renderDetailView() string {
	width := m.width

	var cpuHist, memHist []float64
	if m.detailCPUHistory != nil {
		cpuHist = m.detailCPUHistory.Values()
	}
	if m.detailMemHistory != nil {
		memHist = m.detailMemHistory.Values()
	}

	detailView := panels.RenderProcessDetail(m.detailInfo, cpuHist, memHist, width, m.height)
	help := panels.RenderHelp(m.showHelp, true)

	return RenderDetailLayout(width, m.height, detailView, help)
}

// renderReplayHeader renders the replay-mode header with timeline indicator.
func (m Model) renderReplayHeader(snap metrics.Snapshot) string {
	nowTS := time.Now().UnixMilli()
	var replayTS int64
	if len(m.replayTimestamps) > 0 && m.replayIndex < len(m.replayTimestamps) {
		replayTS = m.replayTimestamps[m.replayIndex]
	}

	offset := history.FormatReplayOffset(replayTS, nowTS)
	timeStr := snap.Timestamp.Format("15:04:05")

	// Build timeline bar.
	barWidth := 20
	pos := 0
	if len(m.replayTimestamps) > 1 {
		pos = m.replayIndex * (barWidth - 1) / (len(m.replayTimestamps) - 1)
	}
	bar := make([]byte, barWidth)
	for i := range bar {
		if i == pos {
			bar[i] = '#'
		} else {
			bar[i] = '-'
		}
	}

	return fmt.Sprintf("  REPLAY  %s  [%s]  %s   Press Esc to exit", offset, string(bar), timeStr)
}

// tickCmd sends a tick after the given refresh interval.
func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// collectSystemCmd runs system metrics collection in a goroutine.
func collectSystemCmd(c *metrics.Collector) tea.Cmd {
	return func() tea.Msg {
		sys, err := c.CollectSystem()
		if err != nil {
			return errMsg(err)
		}
		return systemMsg(sys)
	}
}

// collectProcessesCmd runs process metrics collection in a goroutine.
func collectProcessesCmd(c *metrics.Collector) tea.Cmd {
	return func() tea.Msg {
		procs, _ := c.CollectProcesses()
		return processMsg(procs)
	}
}

// collectConnCountsCmd collects network connection counts per PID.
func collectConnCountsCmd() tea.Cmd {
	return func() tea.Msg {
		return connCountMsg(metrics.CollectConnectionCounts())
	}
}

// collectDetailCmd collects detailed info for a single process.
func collectDetailCmd(pid int32, connCount int) tea.Cmd {
	return func() tea.Msg {
		detail, err := metrics.CollectProcessDetail(pid, connCount)
		return processDetailMsg{detail: detail, err: err}
	}
}

// detailTickCmd sends a tick for refreshing the detail view.
func detailTickCmd() tea.Cmd {
	return tea.Tick(detailTickInterval, func(t time.Time) tea.Msg {
		return detailTickMsg(t)
	})
}

// exportJSONCmd exports the current snapshot to a JSON file.
func exportJSONCmd(snap metrics.Snapshot, connCounts map[int32]int) tea.Cmd {
	return func() tea.Msg {
		path := metrics.ExportPath("", "json")
		written, err := metrics.ExportJSON(snap, connCounts, path)
		return exportDoneMsg{path: written, err: err}
	}
}

// csvWriteCmd appends snapshot data to a CSV file.
func csvWriteCmd(snap metrics.Snapshot, connCounts map[int32]int, path string, writeHeader bool) tea.Cmd {
	return func() tea.Msg {
		err := metrics.AppendCSV(snap, connCounts, path, writeHeader)
		return csvWriteDoneMsg{err: err}
	}
}

// statusClearCmd clears the status message after the given duration.
func statusClearCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return statusClearMsg{}
	})
}
