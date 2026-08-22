#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib/paths.sh"

OUT_JSON="${1:?targets.json path required}"
WANT="${2:-parser,mongo,telegram}"
STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
SAMPLE_RATE="${PARSER_BPF_SAMPLE_RATE:-1}"
NATIVE="${PARSER_BPF_NATIVE:-0}"

mkdir -p "$(dirname "$OUT_JSON")"

role_parser=1
role_telegram=2
role_mongo=3
role_loadgen=4
role_worker=5

entries=""
seen_pids=""

cgroup_id_for_pid() {
  local pid=$1
  local rel path
  # BPF cgroup_id key = inode of the process cgroup v2 directory (see targets.json).
  rel="$(awk -F: '$1=="0" {gsub(/^\/+/, "", $2); print $2}' "/proc/${pid}/cgroup" 2> /dev/null || true)"
  if [[ -z "$rel" ]]; then
    printf '0'
    return
  fi
  path="/sys/fs/cgroup/${rel}"
  if [[ ! -d "$path" ]]; then
    path="/sys/fs/cgroup${rel}"
  fi
  stat -c '%i' "$path" 2> /dev/null || printf '0'
}

add_entry() {
  local pid=$1 role=$2 name=$3
  local cgroup_id=0
  if [[ -z "$pid" || "$pid" == "0" ]]; then
    return 0
  fi
  if [[ " $seen_pids " == *" $pid "* ]]; then
    return 0
  fi
  seen_pids+=" $pid"
  cgroup_id="$(cgroup_id_for_pid "$pid")"
  entries+=$(printf '{"pid":%s,"cgroup_id":%s,"role":%s,"name":"%s"},' "$pid" "$cgroup_id" "$role" "$name")
}

resolve_container() {
  local pattern=$1
  local role=$2
  local cid name pid match_pat
  # Anchor bare names so bidshard-parser does not match bidshard-parser-mongo.
  if [[ "$pattern" == ^* ]]; then
    match_pat="$pattern"
  else
    match_pat="^${pattern}$"
  fi
  cid="$(docker ps --format '{{.ID}} {{.Names}}' | awk -v p="$match_pat" '$2 ~ p {print $1; exit}')"
  if [[ -z "$cid" ]]; then
    return 0
  fi
  name="$(docker inspect -f '{{.Name}}' "$cid" | sed 's#^/##')"
  pid="$(docker inspect -f '{{.State.Pid}}' "$cid" 2> /dev/null || echo 0)"
  add_entry "$pid" "$role" "$name"
}

resolve_native_pattern() {
  local pattern=$1
  local role=$2
  local label=$3
  local pid
  pid="$(pgrep -n -f "$pattern" 2> /dev/null || true)"
  if [[ -z "$pid" ]]; then
    return 0
  fi
  add_entry "$pid" "$role" "$label"
}

resolve_native_defaults() {
  if [[ "$WANT" == *parser* ]]; then
    resolve_native_pattern '[c]md/parser' "$role_parser" 'native-parser'
    resolve_native_pattern '/bin/parser' "$role_parser" 'native-parser'
    resolve_native_pattern 'telegram web' "$role_parser" 'native-parser-tgweb'
    resolve_native_pattern 'parser run' "$role_parser" 'native-parser-run'
  fi
  if [[ "$WANT" == *telegram* ]]; then
    resolve_native_pattern 'sources/telegram/scraper' "$role_telegram" 'native-telegram-py'
    resolve_native_pattern 'telethon' "$role_telegram" 'native-telethon'
  fi
  if [[ "$WANT" == *mongo* ]]; then
    resolve_native_pattern '[m]ongod' "$role_mongo" 'native-mongo'
    resolve_native_pattern '[m]ongo' "$role_mongo" 'native-mongo'
  fi
  if [[ "$WANT" == *worker* ]]; then
    resolve_native_pattern 'pipeline' "$role_worker" 'native-worker'
  fi
}

resolve_native_custom() {
  local spec item role pattern
  spec="${PARSER_BPF_NATIVE_PATTERNS:-}"
  [[ -z "$spec" ]] && return 0
  IFS=',' read -r -a items <<< "$spec"
  for item in "${items[@]}"; do
    role="${item%%:*}"
    pattern="${item#*:}"
    [[ -z "$role" || -z "$pattern" || "$pattern" == "$item" ]] && continue
    resolve_native_pattern "$pattern" "$role" "native-${pattern}"
  done
}

parse_extra_targets() {
  local spec item pid role name
  spec="${PARSER_BPF_EXTRA_TARGETS:-}"
  [[ -z "$spec" ]] && return 0
  IFS=',' read -r -a items <<< "$spec"
  for item in "${items[@]}"; do
    pid="${item%%:*}"
    rest="${item#*:}"
    role="${rest%%:*}"
    name="${rest#*:}"
    [[ -z "$pid" || -z "$role" || "$name" == "$rest" ]] && continue
    add_entry "$pid" "$role" "$name"
  done
}

if [[ "$WANT" == *parser* ]]; then
  resolve_container '^parser$' "$role_parser" || true
  resolve_container 'bidshard-parser' "$role_parser" || true
  resolve_container 'lead-intent-processor-parser' "$role_parser" || true
fi
if [[ "$WANT" == *mongo* ]]; then
  resolve_container 'bidshard-parser-mongo' "$role_mongo" || true
  resolve_container 'mongo' "$role_mongo" || true
fi
if [[ "$WANT" == *telegram* ]]; then
  resolve_container 'telegram' "$role_telegram" || true
  resolve_container 'telethon' "$role_telegram" || true
fi

if [[ "$NATIVE" == "1" ]]; then
  resolve_native_defaults
  resolve_native_custom
fi

parse_extra_targets

entries="${entries%,}"
if [[ -z "$entries" ]]; then
  printf 'bpf-resolve-targets: WARN no PIDs found (stack running? set PARSER_BPF_NATIVE=1 or PARSER_BPF_EXTRA_TARGETS)\n' >&2
fi

cat > "$OUT_JSON" << EOF
{
  "started_at": "$STARTED_AT",
  "sample_rate": $SAMPLE_RATE,
  "roles_wanted": "$WANT",
  "targets": [${entries}]
}
EOF

printf 'bpf-resolve-targets: wrote %s\n' "$OUT_JSON"
