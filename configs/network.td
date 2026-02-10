# Network-focused preset
# For debugging connectivity, API traffic, or container networking.
# Network top panel dominates, with process table below.

global {
  refresh = "1s"
  title   = "network"
}

layout {
  row {
    weight = 2
    col {
      weight = 1
      widget = "nettop"
    }
    col {
      weight = 1
      widget = "summary"
    }
  }
  row {
    weight = 3
    col {
      weight = 1
      widget = "procs"
    }
  }
}
