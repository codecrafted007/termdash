# Ops / SysAdmin preset
# Balanced view: system health up top, processes + network below.
# Good for monitoring servers, VMs, or remote hosts.

global {
  refresh          = "2s"
  title            = "ops"
  history          = "7d"
  history_interval = "10s"
  history_procs    = 100
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
    col {
      weight = 1
      widget = "summary"
    }
  }
  row {
    weight = 2
    col {
      weight = 3
      widget = "procs"
    }
    col {
      weight = 2
      widget = "nettop"
    }
  }
}
