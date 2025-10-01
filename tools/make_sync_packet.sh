#!/usr/bin/env bash
set -euo pipefail

OUT=${1:-sync_packet.txt}

{
  echo "# Plum Sync Packet"
  echo
  echo "## Tree (-L 2)"
  tree -L 2 || true
  echo
  echo "## go.mod"
  [ -f controller/go.mod ] && cat controller/go.mod || true
  echo
  echo "## CMakeLists.txt (agent)"
  [ -f agent/CMakeLists.txt ] && cat agent/CMakeLists.txt || true
  echo
  echo "## Diff Name-Status"
  git diff --name-status || true
  echo
  echo "## Diff Stat"
  git diff --stat || true
  echo
  echo "## Key Files (HTTP handlers)"
  if [ -f controller/internal/httpapi/handlers.go ]; then
    sed -n '1,200p' controller/internal/httpapi/handlers.go
  fi
} > "$OUT"

echo "Wrote $OUT"


