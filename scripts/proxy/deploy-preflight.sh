#!/usr/bin/env bash
# Pre-deploy gate: VPS proxy/config checks + tgweb crawl under eBPF leak probe (Linux).
#
# Usage:
#   make deploy-preflight
#   PARSER_BPF_PREFLIGHT=0 make deploy-preflight   # skip BPF (proxy/config only)
#   DOMAINS=blask.com,vietnam.vn make deploy-preflight
#
# BPF artifacts: var/bpf-session/tgweb-<utc>/{bpf/maps/summary.json,bpf-report.md}
# Requires for BPF: Linux, sudo (or PARSER_BPF_SUDO_PASS), mongo, tgweb registry seed.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

log() { printf 'deploy-preflight: %s\n' "$*"; }

bash "$ROOT/scripts/proxy/vps-preflight.sh"

if [[ "${PARSER_BPF_PREFLIGHT:-1}" != "1" ]]; then
	log "SKIP BPF leak gate (PARSER_BPF_PREFLIGHT=0)"
	log "ok deploy preflight passed (proxy/config only)"
	exit 0
fi

if [[ "$(uname -s)" != "Linux" ]]; then
	log "SKIP BPF leak gate (requires Linux; run on staging VPS before ship)"
	log "ok deploy preflight passed (proxy/config only)"
	exit 0
fi

log "running BPF leak gate (tgweb crawl + eBPF probe)"
# shellcheck source=scripts/lib/bpf_collector.sh
source "$ROOT/scripts/lib/bpf_collector.sh"
if ! bpf_ensure_sudo_for_collector "deploy-preflight"; then
	log "hint: sudo make deploy-preflight"
	log "hint: or export PARSER_BPF_SUDO_PASS for non-interactive sudo"
	exit 1
fi
bash "$ROOT/scripts/tgweb/bpf-leak-preflight.sh"

log "ok deploy preflight passed (proxy/config + BPF leak gate)"
