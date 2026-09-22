# Short demo for the README GIF.
# Prerequisites:
#   - npm and git are already wrapped in ~/.zshrc.
#   - TYPESAFE_API_KEY is configured.
#   - ~/jym-demo does not already exist.

Require zsh
Require jym
Require npm
Require git

Output demo/readme-frames/

Set Shell zsh
Set Width 1000
Set Height 720
Set FontSize 24
Set Framerate 15
Set TypingSpeed 18ms
Set Theme "Catppuccin Mocha"

Env GIT_AUTHOR_NAME "jym demo"
Env GIT_AUTHOR_EMAIL "demo@example.com"
Env GIT_COMMITTER_NAME "jym demo"
Env GIT_COMMITTER_EMAIL "demo@example.com"
Env GIT_CONFIG_GLOBAL "/dev/null"
Env NO_COLOR ""
Env JYM_COLOR "always"

Hide
Type "cd ~"
Enter
Type 'eval "$(jym --shell-integration zsh npm git)"'
Enter
Type "export PS1='%F{cyan}❯%f '"
Enter
Type "clear"
Enter
Show
Sleep 500ms

Type "mkdir jym-demo"
Sleep 50ms
Enter
Sleep 150ms

Type "cd jym-demo"
Sleep 50ms
Enter
Sleep 150ms

Type "npm initialize -y --silent"
Enter
Wait+Line /\[Enter\] run/
Sleep 350ms
Enter
Sleep 500ms

Type "git initialize -b main"
Enter
Wait+Line /\[Enter\] run/
Sleep 350ms
Enter
Sleep 500ms

Type "git append package.json"
Enter
Wait+Line /\[Enter\] run/
Sleep 350ms
Enter
Sleep 500ms

Type 'git record -m "Initial commit"'
Enter
Wait+Line /\[Enter\] run/
Sleep 350ms
Enter
Sleep 500ms

Type "git log --oneline -1"
Enter
Sleep 1s
