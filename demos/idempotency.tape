Output demos/idempotency.gif

Set Shell "bash"
Set FontSize 15
Set Width 1000
Set Height 480
Set Theme "Catppuccin Mocha"
Set WindowBar Colorful
Set BorderRadius 8
Set Margin 20
Set MarginFill "#1e1e2e"
Set Padding 12
Set Framerate 24
Set TypingSpeed 50ms

Hide
Type "export PS1='$ ' && clear"
Enter
Sleep 300ms
Show

Type "bash demos/scripts/idempotency.sh"
Sleep 200ms
Enter
Sleep 3500ms

Sleep 1000ms
