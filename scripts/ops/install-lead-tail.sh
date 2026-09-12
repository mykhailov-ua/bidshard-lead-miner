#!/usr/bin/env bash
# Install lead-tail into ~/.local/bin (one command from anywhere).
#
# Usage:
#   make install-lead-tail
#   bash scripts/ops/install-lead-tail.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BIN_DIR="${LEAD_TAIL_BIN_DIR:-${HOME}/.local/bin}"
MARKER_DIR="${HOME}/.config/lead-intent-processor"
MARKER_FILE="${MARKER_DIR}/root"
chmod +x "${ROOT}/scripts/ops/lead-tail" "${ROOT}/scripts/ops/lead-logs"

mkdir -p "$BIN_DIR" "$MARKER_DIR"
ln -sf "${ROOT}/scripts/ops/lead-tail" "${BIN_DIR}/lead-tail"
ln -sf "${ROOT}/scripts/ops/lead-logs" "${BIN_DIR}/lead-logs"
printf '%s\n' "$ROOT" >"$MARKER_FILE"

if [[ -x "${ROOT}/bin/crm-bot" ]]; then
	:
elif command -v go >/dev/null 2>&1; then
	printf 'install-lead-tail: building bin/crm-bot...\n'
	( cd "$ROOT" && go build -o bin/crm-bot ./cmd/crm-bot )
else
	printf 'install-lead-tail: warn: go not found; install Go or run make build-crm-bot\n' >&2
fi

printf 'install-lead-tail: linked %s/lead-tail\n' "$BIN_DIR"
printf 'install-lead-tail: linked %s/lead-logs\n' "$BIN_DIR"
printf 'install-lead-tail: repo marker %s\n' "$MARKER_FILE"

case ":${PATH}:" in
*":${BIN_DIR}:"*) ;;
*)
	printf 'install-lead-tail: add to PATH (zsh):\n  export PATH="%s:$PATH"\n' "$BIN_DIR"
	;;
esac

printf 'install-lead-tail: run from anywhere:\n  lead-tail\n  lead-logs\n'
