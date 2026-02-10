# Code Flow: Two-Tier Async Parallel Metrics Collection

## BEFORE (Sequential, Single Message)

```
tickMsg (every 2s)
  │
  ▼
collectCmd(collector)          ── single goroutine ──
  │
  ├─ collectCPU()              ~1ms
  ├─ collectMemory()           ~1ms
  ├─ collectDisks()            ~1ms
  ├─ collectNetwork()          ~1ms
  ├─ collectHostInfo()         ~1ms
  ├─ procCollector.CollectAll() ~50-500ms  ◄── BLOCKS everything above
  │
  ▼
snapshotMsg (all-at-once)
  │
  ▼
UI repaints everything ── FLICKER (gap while waiting for processes)
```

## AFTER (Parallel, Two Messages)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          STARTUP (Init)                                 │
│                                                                         │
│  Init() ──► tea.Batch(                                                  │
│               collectSystemCmd(collector),    ◄── goroutine A            │
│               collectProcessesCmd(collector), ◄── goroutine B            │
│             )                                                            │
│                                                                         │
│  Both fire simultaneously, race independently to the Bubble Tea runtime │
└─────────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════

TIER 1: System Metrics (FAST, ~1-5ms total)          TIER 2: Process Metrics (SLOW, ~50-500ms)
─────────────────────────────────────────             ──────────────────────────────────────────

goroutine A: collectSystemCmd                         goroutine B: collectProcessesCmd
  │                                                     │
  ▼                                                     ▼
CollectSystem()                                       CollectProcesses()
  │                                                     │
  ├── goroutine 1: collectCPU()      ─┐                 ▼
  ├── goroutine 2: collectMemory()   ─┤ parallel      procCollector.CollectAll()
  ├── goroutine 3: collectDisks()    ─┤ via              │
  │                  + collectHost() ─┤ WaitGroup        ├─ mu.Lock()  ◄── mutex guards
  │                                   │                  ├─ Pids()        concurrent access
  ├── collectNetwork() ◄── calling   ─┘                  ├─ per-PID collection
  │   goroutine (stateful,             sync.WaitGroup    ├─ sort by CPU%
  │   needs prevBytes*)                .Wait()           ├─ mu.Unlock()
  │                                                      │
  ▼                                                      ▼
systemMsg ◄── arrives FIRST (~5ms)                    processMsg ◄── arrives LATER (~50-500ms)
  │                                                      │
  ▼                                                      ▼

═══════════════════════════════════════════════════════════════════════════

┌─────────────────────────────────────────────────────────────────────────┐
│                     UI Update() Handler                                 │
│                                                                         │
│  case systemMsg:                         case processMsg:               │
│    │                                       │                            │
│    ├─ Update m.snapshot fields:            ├─ Update m.snapshot.        │
│    │   CPU, Memory, Disks, Network,        │   Processes                │
│    │   Hostname, OS, Uptime                │                            │
│    │                                       ├─ Clamp cursor             │
│    ├─ Push to history rings:               │                            │
│    │   cpuHistory, sendHistory,            ├─ Cleanup expanded         │
│    │   recvHistory                         │   groups                   │
│    │                                       │                            │
│    ├─ m.ready = true                       └─ return m, nil            │
│    │   (UI starts rendering)                  (NO tick scheduling)      │
│    │                                                                    │
│    ├─ Schedule tickCmd() ◄── TICK OWNERSHIP: only systemMsg             │
│    │                         schedules the next tick                    │
│    ├─ connTickCounter++ (every 3rd → collectConnCountsCmd)             │
│    │                                                                    │
│    └─ CSV write if logging active                                      │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════

                        TICK LIFECYCLE (steady state)

    ┌───────────────────┐
    │ tickMsg fires     │ (2s after previous systemMsg arrived)
    │ (every ~2s)       │
    └───────┬───────────┘
            │
            ▼
    tea.Batch(
      collectSystemCmd,    ──► goroutine A ──► systemMsg  (~5ms)
      collectProcessesCmd, ──► goroutine B ──► processMsg (~50-500ms)
    )
            │                        │               │
            │                        ▼               │
            │               systemMsg arrives        │
            │                 ├─ UI repaints          │
            │                 │  system panels        │
            │                 │  IMMEDIATELY          │
            │                 │                       ▼
            │                 ├─ tickCmd()     processMsg arrives
            │                 │  scheduled      ├─ UI repaints
            │                 │  (next tick      │  process table
            │                 │   in ~2s)        │  (catches up)
            │                 │                  │
            ▼                 ▼                  ▼

═══════════════════════════════════════════════════════════════════════════

                     FIRST TICK (startup) TIMELINE

    t=0ms     Init() fires both commands
              ├─ collectSystemCmd  → goroutine A starts
              └─ collectProcessesCmd → goroutine B starts

    t=5ms     systemMsg arrives
              ├─ m.ready = true → UI shows system panels
              ├─ Process table empty (no processMsg yet)
              └─ tickCmd() scheduled (fires at t=2005ms)

    t=200ms   processMsg arrives
              ├─ m.snapshot.Processes populated
              └─ Process table renders on next View() call

    t=2005ms  tickMsg fires → both commands fire again
              ... cycle repeats ...

═══════════════════════════════════════════════════════════════════════════

                     CONCURRENCY SAFETY

    CollectSystem() internal parallelism:
    ┌──────────────────────────────────────────────────┐
    │  goroutine 1 ─── collectCPU()     ──┐            │
    │  goroutine 2 ─── collectMemory()  ──┤ WaitGroup  │
    │  goroutine 3 ─── collectDisks()   ──┤            │
    │                   collectHostInfo() ┘            │
    │  calling grtn ─── collectNetwork()  (sequential) │
    │                       │                          │
    │                   wg.Wait() ◄── barrier          │
    └──────────────────────────────────────────────────┘
    No races: each goroutine writes to its own variables.
    Network is stateful (prevBytesSent/Recv) → stays on caller.

    ProcessCollector.CollectAll():
    ┌──────────────────────────────────────────────────┐
    │  pc.mu.Lock()                                    │
    │    ... entire CollectAll body ...                 │
    │  pc.mu.Unlock()  (via defer)                     │
    │                                                  │
    │  If tick overlaps previous process collection,   │
    │  second call blocks on mutex until first finishes│
    └──────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════

                     FILE MAP

    collector.go:98   CollectSystem()    ─── new: parallel system collection
    collector.go:159  CollectProcesses() ─── new: thin wrapper
    collector.go:165  Collect()          ─── refactored: delegates to above two
    process.go:14     ProcessCollector   ─── added: sync.Mutex field
    process.go:30     CollectAll()       ─── added: mu.Lock()/defer mu.Unlock()
    model.go:86       systemMsg          ─── new message type
    model.go:87       processMsg         ─── new message type
    model.go:170      Init()             ─── fires both cmds via tea.Batch
    model.go:187      tickMsg handler    ─── fires both cmds via tea.Batch
    model.go:193      systemMsg handler  ─── updates system fields + owns tick
    model.go:219      processMsg handler ─── updates processes only, no tick
    model.go:831      collectSystemCmd() ─── new tea.Cmd
    model.go:840      collectProcessesCmd() ─── new tea.Cmd
```

## Key Design Decisions

1. **Tick ownership** — Only `systemMsg` schedules `tickCmd()`. Since system metrics complete in ~5ms, the next tick fires ~2s later. Process collection doesn't affect tick cadence.

2. **Network on calling goroutine** — `collectNetwork()` mutates `c.prevBytesSent`/`c.prevBytesRecv`. Keeping it off goroutines avoids needing a mutex on the `Collector` itself.

3. **Mutex on ProcessCollector, not Collector** — Only process collection can overlap (slow collection may still be running when next tick fires). System collection is serialized by tick ownership.

4. **`Collect()` preserved** — Backward compatibility for tests and export. Internally delegates to `CollectSystem()` + `CollectProcesses()`.
