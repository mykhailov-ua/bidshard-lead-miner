#!/usr/bin/env bash
# Run a command under xvfb when PARSER_HEADLESS_HEADED=true or PARSER_HEADLESS_XVFB=1.
set -euo pipefail

use_xvfb=0
if [[ "${PARSER_HEADLESS_XVFB:-}" == "1" ]]; then
	use_xvfb=1
fi
if [[ "${PARSER_HEADLESS_HEADED:-}" == "1" || "${PARSER_HEADLESS_HEADED:-}" == "true" ]]; then
	use_xvfb=1
fi

if [[ "$use_xvfb" == "1" ]]; then
	if ! command -v xvfb-run >/dev/null 2>&1; then
		printf 'headless-xvfb: FAIL xvfb-run not installed (apt install xvfb)\n' >&2
		exit 1
	fi
	exec xvfb-run -a "$@"
fi
exec "$@"
