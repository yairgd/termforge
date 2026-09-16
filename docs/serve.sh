#!/usr/bin/env bash
# Serve termforge documentation locally (uses .venv-docs when present).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
VENV="${ROOT}/.venv-docs"
PY="python3"

if [[ -x "${VENV}/bin/python" ]]; then
  PY="${VENV}/bin/python"
elif ! "${PY}" -m mkdocs --version >/dev/null 2>&1; then
  echo "MkDocs not found. Run: ./docs/setup-docs-venv.sh" >&2
  exit 1
fi

# 8775/8776 are termforge's ports; gdbforge uses 8765/8766 so both can serve at once.
exec "${PY}" -m mkdocs serve \
  --dev-addr "${MKDOCS_DEV_ADDR:-127.0.0.1:8775}" \
  "$@"
