#!/bin/zsh

if [ $# -lt 1 ]; then
  echo "usage: $(dirname "$0")" ITERM_SHELL_INTEGRATION_SCRIPT >&2
  exit 1
fi

iterm_shell_integration_script="$1"
dotdir=$(/usr/bin/mktemp -d itermctl-test-zdotdir.XXXXXXXX)

cat > "$dotdir/.zshrc" <<EOF
source $iterm_shell_integration_script
EOF

export ZDOTDIR="$dotdir"
zsh -i --login
