#!/bin/sh
#
# Container entrypoint: fix data dir ownership, exec parser as non-root.
#
# Usage: automatic via Dockerfile ENTRYPOINT; CMD is the parser subcommand.
# Requires: parser user (UID 10001) and su-exec in the image.
#
set -e

# Bind mounts and named volumes may be root-owned; Telethon/SQLite need write access.
mkdir -p /app/data/runtime /app/data/export
chown -R parser:parser /app/data/runtime /app/data/export

# Drop to parser user before exec; CMD passes subcommand (run, scan, telegram, ...).
exec su-exec parser parser "$@"
