#!/usr/bin/env bash
# Restore parser MongoDB from archive created by backup-mongo.sh.
# Usage: ./scripts/ops/restore-mongo.sh backups/mongo-YYYYMMDD-HHMMSS/dump.gz
set -euo pipefail

if [[ $# -lt 1 ]]; then
	echo "usage: $0 <dump.gz>" >&2
	exit 1
fi

DUMP="$1"
DIR="$(cd "$(dirname "$0")/../.." && pwd)"

if [[ -f "$DIR/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$DIR/.env"
	set +a
fi

MONGO_URI="${MONGO_URI:-mongodb://localhost:27017}"
MONGO_DB="${MONGO_DB:-parser}"

if [[ ! -f "$DUMP" ]]; then
	echo "error: file not found: $DUMP" >&2
	exit 1
fi

echo "Restoring $DUMP into db=$MONGO_DB ..."

if command -v mongorestore >/dev/null 2>&1; then
	gunzip -c "$DUMP" | mongorestore --uri="$MONGO_URI" --nsInclude="${MONGO_DB}.*" --archive --gzip --drop
elif [[ -n "${MONGO_CONTAINER:-}" ]]; then
	gunzip -c "$DUMP" | docker exec -i "$MONGO_CONTAINER" mongorestore --nsInclude="${MONGO_DB}.*" --archive --gzip --drop
else
	echo "error: mongorestore not found in PATH" >&2
	echo "  install mongodb-database-tools, or set MONGO_CONTAINER=<docker mongo name>" >&2
	exit 1
fi

echo "Restore complete."
