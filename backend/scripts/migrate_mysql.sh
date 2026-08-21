#!/usr/bin/env bash
# Dumps the source MySQL database to a timestamped file, then restores it
# onto the destination host. Requires: mysqldump, mysql (client).
#
# Usage:
#   source scripts/migrate.env   # or export the SRC_/DST_ vars yourself
#   ./scripts/migrate_mysql.sh              # dump + restore
#   ./scripts/migrate_mysql.sh --dump-only  # only produce the dump file
#   ./scripts/migrate_mysql.sh --restore-only path/to/dump.sql  # restore an existing dump
set -euo pipefail

for v in SRC_MYSQL_HOST SRC_MYSQL_PORT SRC_MYSQL_USER SRC_MYSQL_DATABASE; do
  [[ -n "${!v:-}" ]] || { echo "Missing required env var: $v" >&2; exit 1; }
done

MODE="${1:-full}"
BACKUP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/backups"
mkdir -p "$BACKUP_DIR"
STAMP="$(date +%Y%m%d_%H%M%S)"
DUMP_FILE="$BACKUP_DIR/${SRC_MYSQL_DATABASE}_${STAMP}.sql"

dump() {
  echo "==> Dumping ${SRC_MYSQL_DATABASE} from ${SRC_MYSQL_HOST}:${SRC_MYSQL_PORT} to ${DUMP_FILE}"
  mysqldump \
    --host="${SRC_MYSQL_HOST}" \
    --port="${SRC_MYSQL_PORT}" \
    --user="${SRC_MYSQL_USER}" \
    ${SRC_MYSQL_PASSWORD:+--password="${SRC_MYSQL_PASSWORD}"} \
    --single-transaction \
    --routines \
    --triggers \
    --set-gtid-purged=OFF \
    "${SRC_MYSQL_DATABASE}" > "${DUMP_FILE}"
  echo "==> Dump complete ($(du -h "${DUMP_FILE}" | cut -f1))"
}

restore() {
  local file="$1"
  for v in DST_MYSQL_HOST DST_MYSQL_PORT DST_MYSQL_USER DST_MYSQL_DATABASE; do
    [[ -n "${!v:-}" ]] || { echo "Missing required env var: $v" >&2; exit 1; }
  done
  echo "==> Restoring ${file} into ${DST_MYSQL_DATABASE} on ${DST_MYSQL_HOST}:${DST_MYSQL_PORT}"
  mysql \
    --host="${DST_MYSQL_HOST}" \
    --port="${DST_MYSQL_PORT}" \
    --user="${DST_MYSQL_USER}" \
    ${DST_MYSQL_PASSWORD:+--password="${DST_MYSQL_PASSWORD}"} \
    -e "CREATE DATABASE IF NOT EXISTS \`${DST_MYSQL_DATABASE}\`;"
  mysql \
    --host="${DST_MYSQL_HOST}" \
    --port="${DST_MYSQL_PORT}" \
    --user="${DST_MYSQL_USER}" \
    ${DST_MYSQL_PASSWORD:+--password="${DST_MYSQL_PASSWORD}"} \
    "${DST_MYSQL_DATABASE}" < "${file}"
  echo "==> Restore complete"
}

case "$MODE" in
  --dump-only)
    dump
    ;;
  --restore-only)
    file="${2:?Usage: $0 --restore-only path/to/dump.sql}"
    restore "$file"
    ;;
  full)
    dump
    restore "$DUMP_FILE"
    ;;
  *)
    echo "Unknown mode: $MODE (expected: full, --dump-only, --restore-only <file>)" >&2
    exit 1
    ;;
esac
