#!/usr/bin/env bash
#
# CI guard: forbidden slop patterns (LinkedIn, panic stubs, LLM typography).
#
# Usage:
#   bash scripts/ci/check_parser_slop.sh
#
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

# LLM typography in prod Go (comments, logs, CLI strings). Cyrillic locale fixtures still allowed.
ai_typography_matches="$(rg -n '[—–→←⇒⇐…●─│≥≤≈µΔ“”‘’]' \
	--glob '*.go' \
	--glob '!*_test.go' \
	. 2>/dev/null || true)"
if [[ -n "$ai_typography_matches" ]]; then
	echo "FAIL: non-ASCII slop typography in Go (use ASCII - see .cursor/rules/ai-slop.mdc):"
	echo "$ai_typography_matches"
	violations=$((violations + 1))
fi

ai_typography_py="$(rg -n '[—–→←⇒⇐…●─│≥≤≈µΔ“”‘’]' \
	--glob '*.py' \
	--glob '!test_*.py' \
	sources/telegram/ 2>/dev/null || true)"
if [[ -n "$ai_typography_py" ]]; then
	echo "FAIL: non-ASCII slop typography in Python sidecar:"
	echo "$ai_typography_py"
	violations=$((violations + 1))
fi

# Docs markdown (README, docs/*.md) - no slop typography in prose or tables
doc_typography_matches="$(rg -n '[—–→←⇒⇐…●─│≥≤≈µΔ“”‘’·]' \
	README.md docs/ config/env/ 2>/dev/null || true)"
if [[ -n "$doc_typography_matches" ]]; then
	echo "FAIL: non-ASCII slop typography in docs:"
	echo "$doc_typography_matches"
	violations=$((violations + 1))
fi

ai_typography_sh="$(rg -n '[—–→←⇒⇐…●─│≥≤≈µΔ“”‘’·]' \
	--glob '*.sh' \
	scripts/dev scripts/perf scripts/proxy scripts/vps-proxy scripts/lib scripts/tgweb docker-entrypoint.sh 2>/dev/null || true)"
if [[ -n "$ai_typography_sh" ]]; then
	echo "FAIL: non-ASCII slop typography in shell scripts:"
	echo "$ai_typography_sh"
	violations=$((violations + 1))
fi

if [[ $violations -gt 0 ]]; then
	exit 1
fi

echo "OK: no slop patterns found"
