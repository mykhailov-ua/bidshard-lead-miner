#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

violations=0

check_pattern() {
	local pattern="$1"
	local desc="$2"
	local matches

	matches="$(grep -rn \
		--include='*.go' \
		--exclude='*_test.go' \
		-E "$pattern" \
		. 2>/dev/null || true)"

	if [[ -n "$matches" ]]; then
		echo "FAIL: forbidden $desc:"
		echo "$matches"
		violations=$((violations + 1))
	fi
}

check_pattern 'linkedin\.com/in' 'linkedin.com/in'
check_pattern 'parseLinkedIn' 'parseLinkedIn'
check_pattern 'panic\("not implemented"\)' 'panic("not implemented") in non-test code'

if [[ $violations -gt 0 ]]; then
	exit 1
fi

echo "OK: no slop patterns found"
