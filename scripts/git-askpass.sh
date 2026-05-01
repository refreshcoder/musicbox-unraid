#!/usr/bin/env sh

case "$1" in
  *Username*)
    printf "%s" "x-access-token"
    ;;
  *Password*)
    printf "%s" "${GITHUB_TOKEN}"
    ;;
  *)
    printf "%s" ""
    ;;
esac

