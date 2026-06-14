# Demo tape for wiki. Rendered with ascii-gif (github.com/tamnd/ascii-gif),
# which supplies the window chrome and theme; this file is just the action.
#
#   ascii-gif render docs/demo/wiki.tape -o docs/static/demo.gif
#
# wiki must be on PATH inside the recording shell.

Hide
Type "export PS1='$ ' PATH=/Users/apple/bin:$PATH"
Enter
Type "clear"
Enter
Show

Sleep 500ms
Type "wiki search 'black hole' -n 4"
Sleep 800ms
Enter
Sleep 3s

Type "wiki info Pi"
Sleep 800ms
Enter
Sleep 2.5s

Type "wiki search physics -n 5 -o jsonl | jq -r .title"
Sleep 800ms
Enter
Sleep 3s
