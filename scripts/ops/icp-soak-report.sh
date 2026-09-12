#!/usr/bin/env bash
# P1-11 ICP soak report: export breakdown + acceptance gates + optional VPS metrics.
#
# Usage:
#   make icp-soak-report
#   LEADS_JSONL=data/export/leads.jsonl bash scripts/ops/icp-soak-report.sh
#   bash scripts/ops/icp-soak-report.sh --vps   # pull export + metrics from VPS
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

FROM_VPS=0
SOAK_HOURS="${SOAK_HOURS:-168}"
VPS_METRICS=""
ALERT_LOG_TEXT=""

if [[ "${1:-}" == "--vps" ]]; then
	FROM_VPS=1
fi

if [[ "$FROM_VPS" == 1 ]]; then
	# shellcheck disable=SC1091
	source "$ROOT/scripts/lib/vps_ssh.sh"
	vps_load_config "$ROOT"
	if ! vps_ssh_require "$ROOT"; then
		exit 1
	fi
	mkdir -p "$ROOT/data/export"
	vps_rsync_pull_export "$ROOT" "$ROOT/data/export/vps-leads.jsonl"
	LEADS_JSONL="$ROOT/data/export/vps-leads.jsonl"
	printf 'icp-soak-report: pulled %s\n' "$LEADS_JSONL"
	printf '\nicp-soak-report: VPS metrics snapshot\n'
	VPS_METRICS="$(vps_ssh "curl -sf http://127.0.0.1:9465/metrics 2>/dev/null" || true)"
	ALERT_LOG_TEXT="$(vps_ssh "cd '${VPS_REMOTE_DIR:-/opt/lead-intent-processor}' && docker compose -f docker-compose.telegram-realtime.yaml logs parser-telegram-realtime --since '${SOAK_HOURS}h' 2>&1" || true)"
	if [[ -z "$VPS_METRICS" ]]; then
		printf 'metrics: unavailable (PARSER_METRICS_ADDR?)\n'
	else
		printf '%s\n' "$VPS_METRICS" | grep -E '^(parser_leads_written_total|parser_warm_analysis_failed_total|parser_processor_reject_total|parser_proxy_cooldown_wait_total|parser_proxy_cf_block_total|parser_telethon_sidecar_failed_total)' || true
	fi
else
	if [[ -f "$ROOT/.env" ]]; then
		set -a
		# shellcheck disable=SC1091
		source "$ROOT/.env"
		set +a
	fi
	# shellcheck source=scripts/lib/host_paths.sh
	source "$ROOT/scripts/lib/host_paths.sh"
	apply_host_export_paths "$ROOT"
	LEADS_JSONL="${LEADS_JSONL:-${PARSER_EXPORT_JSON_HOST:-$ROOT/data/export/leads.jsonl}}"
	METRICS_URL="${PARSER_METRICS_URL:-http://127.0.0.1:9465/metrics}"
	VPS_METRICS="$(curl -sf "$METRICS_URL" 2>/dev/null || true)"
fi

# shellcheck source=scripts/lib/soak_gate.sh
source "$ROOT/scripts/lib/soak_gate.sh"

if ! command -v jq >/dev/null 2>&1; then
	printf 'icp-soak-report: FAIL jq required\n' >&2
	exit 1
fi

if [[ ! -s "$LEADS_JSONL" ]]; then
	printf 'icp-soak-report: FAIL missing export: %s\n' "$LEADS_JSONL" >&2
	printf 'hint: run parser 2h+ on bidshard profile or use --vps\n' >&2
	exit 1
fi

printf '\nicp-soak-report: file=%s rows=%s\n' \
	"$LEADS_JSONL" "$(jq -s 'length' "$LEADS_JSONL")"

printf '\n--- M9 observability ---\n'
print_soak_source_breakdown "$LEADS_JSONL" 15
print_soak_contact_valid_rate "$LEADS_JSONL" "${SOAK_MIN_CONTACT_VALID_PCT:-95}" || true
print_soak_synthetic_reject_rate "$VPS_METRICS"
print_soak_alert_per_day "$ALERT_LOG_TEXT" "$SOAK_HOURS"

printf '\n--- source breakdown (raw source id, top 15) ---\n'
jq -s '
  [group_by(.source)[] | {source: .[0].source, n: length}]
  | sort_by(-.n) | .[:15][] | "\(.source)\t\(.n)"
' -r "$LEADS_JSONL" 2>/dev/null || true

printf '\n--- priority by intent source family ---\n'
jq -s '
  def intent: select(
    .source == "forum" or .source == "reddit" or .source == "github"
    or (.source|startswith("forum:")) or (.source|startswith("telegram"))
    or (.source|startswith("github:")) or (.source|startswith("reddit:"))
    or .source == "webpain" or (.source|startswith("webpain:"))
  );
  {
    intent_high: ([.[] | intent | select(.priority=="High")] | length),
    intent_medium: ([.[] | intent | select(.priority=="Medium")] | length),
    supply_rows: ([.[] | select(.source=="supply" or (.source|startswith("supply:")))] | length),
    jobboard_rows: ([.[] | select(.source|startswith("jobboard"))] | length)
  }
' "$LEADS_JSONL"

printf '\n--- telegram pain radar ---\n'
jq -s '
  def is_telegram: (.source | startswith("telegram"));
  def has_reachable_contact:
    any(.contacts[]?;
      (type == "object" and (.type == "telegram" or .type == "email"))
      or (type == "string" and (startswith("telegram:") or test("^[^@]+@[^@]+\\.")))
    );
  {
    telegram_leads: ([.[] | select(is_telegram)] | length),
    telegram_with_contact: ([.[] | select(is_telegram and has_reachable_contact)] | length),
    jobboard_accepted: ([.[] | select(.source | startswith("jobboard"))] | length),
    no_reachable_contact: "metrics-only (rejected rows not in export)"
  }
' "$LEADS_JSONL"

if [[ "$FROM_VPS" == 1 ]]; then
	printf '\n--- telegram pain radar metrics (VPS) ---\n'
	if [[ -z "${VPS_METRICS:-}" ]]; then
		printf 'pain metrics: unavailable (PARSER_METRICS_ADDR?)\n'
	else
		printf '%s\n' "$VPS_METRICS" | grep -E \
			'parser_processor_reject_total\{reason="no_reachable_contact"\}|parser_processor_reject_total\{reason="telegram_spam"\}|parser_processor_reject_total\{reason="intel_only"\}|parser_telethon_sidecar_failed_total|parser_leads_accepted_total\{source="telegram' \
			|| printf 'pain metrics: no matching counters (gate may be idle)\n'
	fi
fi

printf '\n--- acceptance gates ---\n'
fail=0
evaluate_acceptance_soak_gates "$LEADS_JSONL" || fail=1

if [[ "$fail" -ne 0 ]]; then
	printf '\nicp-soak-report: FAIL (see soak-gate lines above)\n' >&2
	exit 1
fi

printf '\nicp-soak-report: ok\n'
