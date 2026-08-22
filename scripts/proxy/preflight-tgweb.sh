#!/usr/bin/env bash
#
# Preflight before tgweb crawl: config check, optional proxy, registry, TLS smoke, HTTP smoke fetch.
#
# Usage:
#   make preflight-tgweb
#
# Requires: mongo up, non-empty TELEGRAM_DOMAINS_PATH registry (make tgweb-seed or tgweb-discover).
# Warn-only: GEMINI_API_KEY unset (sync ICP gate skipped).
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if [[ -f "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

if [[ "${DEPLOY_PREFLIGHT_CI:-}" == "1" || "${GITHUB_ACTIONS:-}" == "true" ]]; then
	export PARSER_SEED_PROFILE=dev
	export PARSER_ICP_CLASSIFY_TGWEB=false
	export PARSER_BG_TELEGRAM=false
fi

log() { printf 'preflight-tgweb: %s\n' "$*"; }
warn() { printf 'preflight-tgweb: WARN %s\n' "$*" >&2; }
fail() { printf 'preflight-tgweb: FAIL %s\n' "$*" >&2; exit 1; }

failures=0
note() { log "$1"; }

# 1. Parser config check (keywords, mongo ping, gemini warnings)
note "running parser config check"
if ! go run ./cmd/parser config check; then
	failures=$((failures + 1))
fi

# 2. Proxy (optional)
proxy="${PARSER_PROXY_LIST:-}"
proxy="${proxy//[[:space:]]/}"
if [[ -n "$proxy" ]]; then
	first="${proxy%%,*}"
	note "checking proxy $first"
	if ! bash "$ROOT/scripts/proxy/check-proxy.sh" "$first"; then
		failures=$((failures + 1))
	fi

	proxy_count="$(printf '%s' "$proxy" | tr ',' '\n' | sed '/^[[:space:]]*$/d' | wc -l)"
	if [[ "${proxy_count:-0}" -gt 1 ]]; then
		note "proxy list: ${proxy_count} URLs (round-robin; prefer sticky session per provider for crawl/login)"
	fi
	proxy_user="${first#*://}"
	proxy_user="${proxy_user%%@*}"
	proxy_user="${proxy_user%%:*}"
	if [[ -n "$proxy_user" ]] && printf '%s' "$proxy_user" | grep -qiE 'country|session|sess|geo|sticky|residential'; then
		note "proxy username has geo/session hint (residential providers)"
	else
		warn "proxy username has no geo/session token - set country/session in provider dashboard if CF blocks crawl"
	fi
else
	# Empty proxy list uses direct egress; prefer this on home IP for Cloudflare targets.
	note "proxy skipped (PARSER_PROXY_LIST empty - direct egress, best for home IP + Cloudflare)"
fi

# 3. Telegram domains registry
domains_path="${TELEGRAM_DOMAINS_PATH:-data/runtime/discovered_telegram_domains.json}"
if [[ "$domains_path" != /* ]]; then
	domains_path="$ROOT/$domains_path"
fi

if [[ ! -f "$domains_path" ]]; then
	warn "registry missing: $domains_path"
	warn "hint: run telegram discover/scrape or copy data/runtime/discovered_telegram_domains.json.example"
	failures=$((failures + 1))
else
	count="$(python3 - "$domains_path" <<'PY'
import json, sys
path = sys.argv[1]
try:
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    domains = data.get("domains") or []
    print(len([d for d in domains if (d.get("domain") or "").strip()]))
except Exception:
    print(0)
PY
)"
	if [[ "${count:-0}" -eq 0 ]]; then
		warn "registry empty: $domains_path (0 domains)"
		warn "hint: parser telegram discover / scrape, or telegram domains prune"
		failures=$((failures + 1))
	else
		note "registry ok: $count domain(s) in $domains_path"
	fi
fi

# 4. TLS fingerprint smoke (proxy egress only; curl JA3 != parser uTLS on direct)
tls_url="${PARSER_PREFLIGHT_TLS_URL:-https://tls.peet.ws/api/all}"
if [[ -n "$proxy" ]]; then
	first="${proxy%%,*}"
	note "tls fingerprint smoke $tls_url (via proxy)"
	tls_args=( -fsSL --max-time 20 -x "$first" )
	if tls_body="$(curl "${tls_args[@]}" "$tls_url" 2>/dev/null)"; then
		ja4="$(printf '%s' "$tls_body" | python3 -c '
import json, sys
try:
    d = json.load(sys.stdin)
    print(d.get("ja4") or (d.get("tls") or {}).get("ja4") or "")
except Exception:
    print("")
')"
		if [[ -n "$ja4" ]]; then
			note "  tls smoke ok ja4=${ja4:0:20}..."
		else
			warn "tls smoke JSON missing ja4 (proxy may block introspection endpoints)"
			failures=$((failures + 1))
		fi
	else
		warn "tls fingerprint smoke failed via proxy"
		failures=$((failures + 1))
	fi
else
	note "tls fingerprint smoke skipped (direct egress; parser uses uTLS Chrome on crawl)"
fi

# 5. Optional smoke fetch (direct or via proxy from env)
# Override with PARSER_PREFLIGHT_SMOKE_URL to test a different egress target.
smoke_url="${PARSER_PREFLIGHT_SMOKE_URL:-https://blask.com/about}"
note "smoke fetch $smoke_url"
curl_args=( -fsSL --max-time 25 -o /dev/null -w "  HTTP %{http_code} in %{time_total}s\n" )
if [[ -n "$proxy" ]]; then
	first="${proxy%%,*}"
	if ! curl "${curl_args[@]}" -x "$first" "$smoke_url"; then
		warn "smoke fetch failed via proxy (CF or block? try direct or residential)"
		failures=$((failures + 1))
	fi
else
	if ! curl "${curl_args[@]}" "$smoke_url"; then
		warn "smoke fetch failed (network or block)"
		failures=$((failures + 1))
	fi
fi

# 6. Gemini for tgweb ICP
if [[ "${PARSER_ICP_CLASSIFY_TGWEB:-true}" == "true" ]] && [[ -z "${GEMINI_API_KEY:-}" ]]; then
	fail "GEMINI_API_KEY required when PARSER_ICP_CLASSIFY_TGWEB=true"
fi

if [[ "$failures" -gt 0 ]]; then
	fail "$failures check(s) failed - fix and re-run: make preflight-tgweb"
fi

log "ok - ready for: go run ./cmd/parser telegram web"
