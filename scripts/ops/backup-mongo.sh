#!/usr/bin/env bash
# Backup parser MongoDB database (default: db "parser").
# Requires mongodump (mongodb-database-tools) OR MONGO_CONTAINER for docker exec.
set -euo pipefail

DIR="$(cd "$(dirname "$0")/../.." && pwd)"
BACKUP_ROOT="${BACKUP_ROOT:-$DIR/backups}"
KEEP_DAYS="${KEEP_DAYS:-14}"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT="$BACKUP_ROOT/mongo-$STAMP"

if [[ -f "$DIR/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$DIR/.env"
	set +a
fi

MONGO_URI="${MONGO_URI:-mongodb://localhost:27017}"
MONGO_DB="${MONGO_DB:-parser}"

mkdir -p "$OUT"

echo "Backing up db=$MONGO_DB to $OUT/dump.gz ..."

if command -v mongodump >/dev/null 2>&1; then
	mongodump --uri="$MONGO_URI" --db="$MONGO_DB" --archive --gzip >"$OUT/dump.gz"
elif [[ -n "${MONGO_CONTAINER:-}" ]]; then
	docker exec -i "$MONGO_CONTAINER" mongodump --db="$MONGO_DB" --archive --gzip >"$OUT/dump.gz"
else
	echo "error: mongodump not found in PATH" >&2
	echo "  install mongodb-database-tools, or set MONGO_CONTAINER=<docker mongo name>" >&2
	exit 1
fi

echo "Done: $OUT/dump.gz"
ls -lh "$OUT/dump.gz"

if [[ "$KEEP_DAYS" -gt 0 ]]; then
	find "$BACKUP_ROOT" -maxdepth 1 -type d -name 'mongo-*' -mtime +"$KEEP_DAYS" -print -exec rm -rf {} +
fi
