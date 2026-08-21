#!/usr/bin/env bash
# Resolve Telethon Python and verify sidecar deps before discover / bg worker runs.

resolve_telethon_python() {
	local root="${1:?root required}"
	if [[ -n "${PARSER_TELETHON_PYTHON:-}" && -x "$PARSER_TELETHON_PYTHON" ]]; then
		printf '%s\n' "$PARSER_TELETHON_PYTHON"
		return 0
	fi
	if [[ -x "$root/.venv/bin/python" ]]; then
		printf '%s\n' "$root/.venv/bin/python"
		return 0
	fi
	printf '%s\n' "python3"
}

telethon_import_ok() {
	local py="${1:?python required}"
	PYTHONPATH="${2:?root required}" "$py" -c 'import telethon' 2>/dev/null
}

telethon_session_path() {
	local root="${1:?root required}"
	local cfg="${TELEGRAM_CONFIG_PATH:-config/sources.telegram.yaml}"
	if [[ "$cfg" != /* ]]; then
		cfg="$root/$cfg"
	fi
	if [[ ! -f "$cfg" ]]; then
		return 1
	fi
	local rel
	rel="$(awk '/^session:[[:space:]]*/ {print $2; exit}' "$cfg")"
	if [[ -z "$rel" ]]; then
		return 1
	fi
	if [[ "$rel" == /* ]]; then
		printf '%s\n' "$rel"
	else
		printf '%s\n' "$root/$rel"
	fi
}

# Export PARSER_TELETHON_PYTHON. Fail when TELEGRAM_API_* set but telethon or session missing.
require_telethon_ready() {
	local root="${1:?root required}"
	local py
	py="$(resolve_telethon_python "$root")"
	export PARSER_TELETHON_PYTHON="$py"

	if [[ -z "${TELEGRAM_API_ID:-}" || -z "${TELEGRAM_API_HASH:-}" ]]; then
		return 0
	fi

	if ! telethon_import_ok "$py" "$root"; then
		printf 'telethon-preflight: FAIL telethon not importable with %s\n' "$py" >&2
		printf 'telethon-preflight: run: make venv\n' >&2
		printf 'telethon-preflight: optional: export PARSER_TELETHON_PYTHON=%s/.venv/bin/python\n' "$root" >&2
		return 1
	fi

	local session
	if ! session="$(telethon_session_path "$root")"; then
		printf 'telethon-preflight: WARN session path not found in %s\n' "${TELEGRAM_CONFIG_PATH:-config/sources.telegram.yaml}" >&2
		printf 'telethon-preflight: run: go run ./cmd/parser telegram login --qr\n' >&2
		return 1
	fi
	if [[ ! -f "$session" ]]; then
		printf 'telethon-preflight: FAIL session missing: %s\n' "$session" >&2
		printf 'telethon-preflight: run: go run ./cmd/parser telegram login --qr\n' >&2
		return 1
	fi

	printf 'telethon-preflight: ok python=%s session=%s\n' "$py" "$session"
	if [[ -n "${TELEGRAM_PROXY_URL:-}" ]]; then
		printf 'telethon-preflight: ok MTProto proxy TELEGRAM_PROXY_URL set\n'
	fi
	return 0
}
