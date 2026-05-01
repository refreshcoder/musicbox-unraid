#!/usr/bin/env sh

case "$1" in
  *Username*)
    printf "%s" "refreshcoder"
    ;;
  *Password*)
    printf "%s" "${GITHUB_TOKEN}"
    ;;
  *)
    printf "%s" ""
    ;;
esac
