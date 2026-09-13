#!/usr/bin/env bash
# Sample residential proxies; failover to home tunnel when 403 ratio is too high.
# Intended for VPS parser host (systemd timer). See config/env/.env.proxy-egress-failover.example
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

STATE_DIR="${PROXY_FAILOVER_STATE_DIR:-$ROOT/var/proxy-egress}"
MODE_FILE="$STATE_DIR/mode"
STREAK_FILE="$STATE_DIR/streak"
LOG_FILE="$STATE_DIR/failover.log"
RESIDENTIAL_SNAPSHOT="${PROXY_FAILOVER_RESIDENTIAL_SNAPSHOT:-$STATE_DIR/residential.snapshot}"
HOME_CRED="${PROXY_FAILOVER_HOME_CRED:-$ROOT/var/home-egress.credentials}"
ENV_FILE="${PROXY_FAILOVER_ENV_FILE:-$ROOT/.env}"
FAILOVER_ENV="${PROXY_FAILOVER_CONFIG:-$STATE_DIR/failover.env}"

mkdir -p "$STATE_DIR"

if [[ -f "$FAILOVER_ENV" ]]; then
	set -a
	# shellcheck disable=SC1090
	source "$FAILOVER_ENV"
	set +a
fi

MIN_OK_RATIO="${PROXY_FAILOVER_MIN_OK_RATIO:-0.55}"
SAMPLE_PROXIES="${PROXY_FAILOVER_SAMPLE_PROXIES:-6}"
PROBE_URLS="${PROXY_FAILOVER_PROBE_URLS:-https://example.com/ https://blask.com/about}"
CONSECUTIVE_BAD="${PROXY_FAILOVER_CONSECUTIVE_BAD:-2}"
CONSECUTIVE_GOOD="${PROXY_FAILOVER_CONSECUTIVE_GOOD:-3}"
HOME_MARK="${PROXY_FAILOVER_HOME_MARK:-127.0.0.1:19888}"
DRY_RUN="${PROXY_FAILOVER_DRY_RUN:-0}"
RESIDENTIAL_LIST_OVERRIDE="${PROXY_FAILOVER_RESIDENTIAL_LIST_FILE:-}"

log() {
	local line="proxy-egress-failover: $*"
	echo "$line"
	printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$line" >>"$LOG_FILE"
}

read_mode() {
	if [[ -f "$MODE_FILE" ]]; then
		cat "$MODE_FILE"
		return 0
	fi
	if [[ -f "$ENV_FILE" ]] && grep -q "$HOME_MARK" "$ENV_FILE" 2>/dev/null; then
		echo home
		return 0
	fi
	echo residential
}

write_mode() {
	printf '%s\n' "$1" >"$MODE_FILE"
}

read_streak() {
	local key="$1"
	if [[ -f "$STREAK_FILE" ]]; then
		# shellcheck disable=SC1090
		source "$STREAK_FILE"
	fi
	case "$key" in
	bad) echo "${bad_streak:-0}" ;;
	good) echo "${good_streak:-0}" ;;
	*) echo 0 ;;
	esac
}

write_streak() {
	local bad="${1:-0}" good="${2:-0}"
	cat >"$STREAK_FILE" <<EOF
bad_streak=$bad
good_streak=$good
EOF
}

is_home_proxy_url() {
	local url="$1"
	[[ "$url" == *"$HOME_MARK"* ]]
}

save_residential_snapshot_if_needed() {
	if [[ -f "$RESIDENTIAL_SNAPSHOT" ]]; then
		return 0
	fi
	if [[ ! -f "$ENV_FILE" ]]; then
		return 0
	fi
	local list_line file_line
	list_line="$(grep -m1 '^PARSER_PROXY_LIST=' "$ENV_FILE" 2>/dev/null || true)"
	file_line="$(grep -m1 '^PARSER_PROXY_LIST_FILE=' "$ENV_FILE" 2>/dev/null || true)"
	if [[ -n "$list_line" ]] && [[ "$list_line" == *"$HOME_MARK"* ]]; then
		log "no residential.snapshot and .env is home; set PROXY_FAILOVER_RESIDENTIAL_LIST_FILE or run vps-save-residential-proxy"
		return 1
	fi
	{
		[[ -n "$list_line" ]] && printf '%s\n' "$list_line"
		[[ -n "$file_line" ]] && printf '%s\n' "$file_line"
		grep -m1 '^PARSER_PROXY_SOURCES=' "$ENV_FILE" 2>/dev/null || true
	} >"$RESIDENTIAL_SNAPSHOT"
	chmod 600 "$RESIDENTIAL_SNAPSHOT"
	log "wrote initial $RESIDENTIAL_SNAPSHOT from .env"
}

resolve_list_path() {
	local rel="$1"
	if [[ "$rel" == /* ]]; then
		printf '%s' "$rel"
	else
		printf '%s/%s' "$ROOT" "$rel"
	fi
}

collect_residential_proxy_urls() {
	local -a urls=()
	local list_file="" line url

	if [[ -n "$RESIDENTIAL_LIST_OVERRIDE" ]]; then
		list_file="$(resolve_list_path "$RESIDENTIAL_LIST_OVERRIDE")"
	elif [[ -f "$RESIDENTIAL_SNAPSHOT" ]]; then
		line="$(grep -m1 '^PARSER_PROXY_LIST_FILE=' "$RESIDENTIAL_SNAPSHOT" 2>/dev/null | cut -d= -f2- | tr -d '\r\"' || true)"
		if [[ -n "$line" ]]; then
			list_file="$(resolve_list_path "$line")"
		fi
		url="$(grep -m1 '^PARSER_PROXY_LIST=' "$RESIDENTIAL_SNAPSHOT" 2>/dev/null | cut -d= -f2- | tr -d '\r\"' || true)"
		if [[ -n "$url" ]] && ! is_home_proxy_url "$url"; then
			urls+=("$url")
		fi
	fi

	if [[ -n "$list_file" && -f "$list_file" ]]; then
		while IFS= read -r line; do
			line="${line%%#*}"
			line="${line//[[:space:]]/}"
			[[ -z "$line" ]] && continue
			if [[ "$line" == http://* || "$line" == https://* ]]; then
				urls+=("$line")
			else
				urls+=("http://$line")
			fi
		done <"$list_file"
	fi

	if [[ ${#urls[@]} -eq 0 ]]; then
		return 1
	fi

	printf '%s\n' "${urls[@]}" | shuf | head -n "$SAMPLE_PROXIES"
}

http_code_via_proxy() {
	local proxy="$1" target="$2"
	local code
	code="$(curl -sS --max-time 22 -x "$proxy" -o /dev/null -w '%{http_code}' "$target" 2>/dev/null || echo 000)"
	printf '%s' "$code"
}

probe_residential_pool() {
	local -a proxies=()
	local proxy url code
	local total=0 ok=0 n403=0 other=0
	local non403 ok_ratio="0"
	local last_probe="$STATE_DIR/last_probe"

	mapfile -t proxies < <(collect_residential_proxy_urls || true)
	if [[ ${#proxies[@]} -eq 0 ]]; then
		log "probe skip: no residential proxy URLs"
		return 2
	fi

	for proxy in "${proxies[@]}"; do
		for url in $PROBE_URLS; do
			code="$(http_code_via_proxy "$proxy" "$url")"
			total=$((total + 1))
			case "$code" in
			403) n403=$((n403 + 1)) ;;
			2?? | 3??) ok=$((ok + 1)) ;;
			*) other=$((other + 1)) ;;
			esac
		done
	done

	non403=$((ok + other))
	if [[ "$total" -gt 0 ]]; then
		ok_ratio="$(awk -v n="$non403" -v t="$total" 'BEGIN { printf "%.4f", n/t }')"
	fi
	log "probe residential: total=$total ok_2xx3xx=$ok http_403=$n403 other=$other non403_ratio=$ok_ratio min_ok_ratio=$MIN_OK_RATIO"
	cat >"$last_probe" <<EOF
total=$total
non403=$non403
ok_ratio=$ok_ratio
EOF
}

home_tunnel_ok() {
	if [[ ! -f "$HOME_CRED" ]]; then
		log "home tunnel check: missing $HOME_CRED"
		return 1
	fi
	set -a
	# shellcheck disable=SC1090
	source "$HOME_CRED"
	set +a
	local url="${PARSER_PROXY_LIST:-}"
	if [[ -z "$url" ]]; then
		log "home tunnel check: empty PARSER_PROXY_LIST in credentials"
		return 1
	fi
	local code
	code="$(http_code_via_proxy "$url" "https://api.ipify.org")"
	if [[ "$code" == "200" ]]; then
		log "home tunnel check: ok (ipify http 200)"
		return 0
	fi
	log "home tunnel check: FAIL ipify http_code=$code"
	return 1
}

merge_env_proxy_lines() {
	local mode="$1"
	local tmp="$ENV_FILE.tmp.$$"
	local sources_line=""

	if [[ ! -f "$ENV_FILE" ]]; then
		log "missing $ENV_FILE"
		return 1
	fi

	if [[ "$mode" == "home" ]]; then
		set -a
		# shellcheck disable=SC1090
		source "$HOME_CRED"
		set +a
		grep -v '^PARSER_PROXY_LIST=' "$ENV_FILE" | grep -v '^PARSER_PROXY_LIST_FILE=' >"$tmp"
		printf 'PARSER_PROXY_LIST=%s\n' "$PARSER_PROXY_LIST" >>"$tmp"
		sources_line="${PARSER_PROXY_SOURCES:-}"
	elif [[ "$mode" == "residential" ]]; then
		if [[ ! -f "$RESIDENTIAL_SNAPSHOT" ]]; then
			log "cannot restore residential: no snapshot"
			return 1
		fi
		grep -v '^PARSER_PROXY_LIST=' "$ENV_FILE" | grep -v '^PARSER_PROXY_LIST_FILE=' >"$tmp"
		if grep -q '^PARSER_PROXY_LIST=' "$RESIDENTIAL_SNAPSHOT"; then
			grep '^PARSER_PROXY_LIST=' "$RESIDENTIAL_SNAPSHOT" >>"$tmp"
		fi
		if grep -q '^PARSER_PROXY_LIST_FILE=' "$RESIDENTIAL_SNAPSHOT"; then
			grep '^PARSER_PROXY_LIST_FILE=' "$RESIDENTIAL_SNAPSHOT" >>"$tmp"
		fi
		sources_line="$(grep -m1 '^PARSER_PROXY_SOURCES=' "$RESIDENTIAL_SNAPSHOT" 2>/dev/null | cut -d= -f2- || true)"
	else
		return 1
	fi

	if [[ -n "$sources_line" ]]; then
		grep -v '^PARSER_PROXY_SOURCES=' "$tmp" >"${tmp}.2"
		mv "${tmp}.2" "$tmp"
		printf 'PARSER_PROXY_SOURCES=%s\n' "$sources_line" >>"$tmp"
	fi

	mv "$tmp" "$ENV_FILE"
	chmod 600 "$ENV_FILE"
}

apply_mode() {
	local mode="$1"
	if [[ "$DRY_RUN" == "1" ]]; then
		log "DRY_RUN: would apply mode=$mode and restart parser"
		write_mode "$mode"
		return 0
	fi
	merge_env_proxy_lines "$mode"
	write_mode "$mode"
	write_streak 0 0
	docker compose restart parser
	log "applied mode=$mode and restarted parser"
}

main() {
	local mode bad good total non403 ok_ratio pass
	local last_probe="$STATE_DIR/last_probe"
	mode="$(read_mode)"
	save_residential_snapshot_if_needed || true

	if ! probe_residential_pool; then
		log "exit: probe not run"
		exit 0
	fi
	# shellcheck disable=SC1090
	source "$last_probe"
	if [[ "${total:-0}" -eq 0 ]]; then
		log "exit: no probe data"
		exit 0
	fi

	pass=0
	if awk -v r="$ok_ratio" -v min="$MIN_OK_RATIO" 'BEGIN { exit (r >= min) ? 0 : 1 }'; then
		pass=1
	fi

	bad="$(read_streak bad)"
	good="$(read_streak good)"

	if [[ "$pass" -eq 1 ]]; then
		good=$((good + 1))
		bad=0
		write_streak "$bad" "$good"
		log "probe pass (streak good=$good need=$CONSECUTIVE_GOOD mode=$mode)"
		if [[ "$mode" == "home" && "$good" -ge "$CONSECUTIVE_GOOD" ]]; then
			log "switching home -> residential (pool recovered)"
			apply_mode residential
		fi
		exit 0
	fi

	bad=$((bad + 1))
	good=0
	write_streak "$bad" "$good"
	log "probe fail (streak bad=$bad need=$CONSECUTIVE_BAD mode=$mode)"

	if [[ "$mode" == "residential" && "$bad" -ge "$CONSECUTIVE_BAD" ]]; then
		if ! home_tunnel_ok; then
			log "stay residential: home tunnel not reachable (keep laptop tunnel up)"
			exit 0
		fi
		log "switching residential -> home (403 ratio above threshold)"
		apply_mode home
	fi
}

main
