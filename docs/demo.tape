Output docs/observe.gif

Set Shell bash
Set FontSize 20
Set Width 1280
Set Height 760
Set Margin 20
Set MarginFill "#0a0e14"
Set Theme "Catppuccin Mocha"
Set TypingSpeed 12ms
Set PlaybackSpeed 1

Hide
Type "go build -o /tmp/observe-demo ."
Enter
Sleep 4s
Show

Type "OBSERVE_NO_ALT_SCREEN=1 /tmp/observe-demo"
Enter
Sleep 4s
Type "2"
Sleep 2.5s
Type "3"
Sleep 2.5s
Type "5"
Sleep 2.5s
Type "1"
Sleep 3s
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "Enter"
Sleep 2.5s
Type "Esc"
Sleep 800ms
Type "?"
Sleep 2s
Type "?"
Sleep 800ms
Type "q"
Enter
Sleep 500ms
Type "/tmp/observe-demo ask is my server cpu bound"
Enter
Sleep 2.5s
