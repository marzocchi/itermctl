#!/bin/zsh

if [ $# -lt 1 ]; then
  echo "usage: $(dirname "$0")" COMMAND >&2
  exit 1
fi

cat ./scripts/itermctl-test-profile.json | sed "s;STARTUP_COMMAND;$1;"
