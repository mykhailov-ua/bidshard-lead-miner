#!/usr/bin/env bash
_SCRIPTS="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT="$(cd "$_SCRIPTS/.." && pwd)"
SCRIPTS="$_SCRIPTS"
export ROOT SCRIPTS
