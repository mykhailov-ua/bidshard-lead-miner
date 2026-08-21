#!/usr/bin/env bash

parser_go_bin() {
  if [[ -n "${PARSER_GO_BIN:-}" && -x "${PARSER_GO_BIN}" ]]; then
    printf '%s' "${PARSER_GO_BIN}"
    return 0
  fi
  local go_bin=""
  go_bin="$(command -v go 2> /dev/null || true)"
  if [[ -z "$go_bin" && -x /usr/local/go/bin/go ]]; then
    go_bin=/usr/local/go/bin/go
  fi
  if [[ -z "$go_bin" && -x "${HOME}/go/bin/go" ]]; then
    go_bin="${HOME}/go/bin/go"
  fi
  if [[ -z "$go_bin" ]]; then
    return 1
  fi
  printf '%s' "$go_bin"
}

parser_go_run() {
  local go_bin
  if ! go_bin="$(parser_go_bin)"; then
    printf 'parser-go: ERROR: go not found (set PARSER_GO_BIN)\n' >&2
    return 127
  fi
  "$go_bin" run "$@"
}

parser_go_build() {
  local go_bin
  if ! go_bin="$(parser_go_bin)"; then
    printf 'parser-go: ERROR: go not found (set PARSER_GO_BIN)\n' >&2
    return 127
  fi
  "$go_bin" build "$@"
}
