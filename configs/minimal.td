# Minimal preset
# Just processes — clean, distraction-free, like a better `top`.
# Slower refresh to keep CPU usage low on laptops.

global {
  refresh = "3s"
  title   = "minimal"
}

layout {
  row {
    weight = 1
    col {
      weight = 1
      widget = "procs"
    }
  }
}
