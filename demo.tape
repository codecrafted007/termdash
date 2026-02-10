# termdash demo recording
# Run: vhs demo.tape

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 800
Set Theme "Catppuccin Mocha"
Set TypingSpeed 80ms
Set Padding 10

# Build and launch termdash
Type "go build -o /tmp/termdash ./cmd/termdash && /tmp/termdash -c configs/dev.td"
Enter
Sleep 4s

# Navigate the process table
Sleep 1s
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "k"
Sleep 400ms

# Cycle sort column
Type "s"
Sleep 1s
Type "s"
Sleep 1s
Type "s"
Sleep 1s

# Sort back to CPU
Type "s"
Sleep 500ms
Type "s"
Sleep 1s

# Open query filter
Type "Q"
Sleep 500ms
Type "cpu > 5"
Sleep 1.5s
Enter
Sleep 2s

# Clear filter
Escape
Sleep 1.5s

# Simple search
Type "/"
Sleep 400ms
Type "chrome"
Sleep 1.5s
Escape
Sleep 1s

# Toggle process grouping
Type "p"
Sleep 2s
Type "p"
Sleep 1s

# Enter process detail view
Type "g"
Sleep 300ms
Enter
Sleep 3s
Escape
Sleep 1.5s

# Enter replay mode
Type "t"
Sleep 2s
Type "["
Sleep 800ms
Type "["
Sleep 800ms
Type "["
Sleep 800ms
Type "]"
Sleep 800ms
Escape
Sleep 2s

# Quit
Type "q"
Sleep 500ms
