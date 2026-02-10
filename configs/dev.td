# Developer preset
# Heavy on process table (spot runaway builds, containers, test runners).
# CPU + Memory side by side at top for quick glance.
# Fast 1s refresh to catch spikes during compile/test cycles.

global {
  refresh          = "1s"
  title            = "dev"
  history          = "24h"
  history_interval = "10s"
  history_procs    = 50
}

layout {
  row {
    weight = 1
    col {
      weight = 1
      widget = "cpu"
    }
    col {
      weight = 1
      widget = "memory"
    }
  }
  row {
    weight = 4
    col {
      weight = 1
      widget = "procs"
    }
  }
}
