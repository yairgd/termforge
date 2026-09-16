#!/usr/bin/env bash
# Fail if the framework depends on a consuming application, on its own example,
# or if the base layer reaches back up into the engine.
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0
imports() {
  go list -f '{{range .Imports}}{{println .}}{{end}}' "$1" 2>/dev/null
}
check() {
  local pkg="$1" bad="$2" why="$3" hits
  hits=$(imports "$pkg" | grep -E "$bad" || true)
  if [[ -n "$hits" ]]; then
    echo "FORBIDDEN ($why): $pkg imports:"
    echo "$hits" | sed 's/^/  /'
    fail=1
  fi
}

# Applications that consume termforge. The framework must never depend on one.
APPS='github.com/yairgd/(gdbforge|stockdash)(/|$)'
for pkg in $(go list ./...); do
  check "$pkg" "$APPS" "framework must not import an application"
done

# The library must not depend on its own example program.
DEMO='github.com/yairgd/termforge/(internal/demo|cmd/demo)(/|$)'
for pkg in $(go list ./... | grep -vE '/(internal/demo|cmd/demo)$'); do
  check "$pkg" "$DEMO" "library must not import the example"
done

# platform is the dependency-free base layer: it must not import the engine
# root or any sibling package.
check ./platform 'github.com/yairgd/termforge(/|$)' "platform is the base layer"

if [[ "$fail" -ne 0 ]]; then
  echo "import guardrails failed"
  exit 1
fi
echo "import guardrails OK"
