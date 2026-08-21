#!/usr/bin/env bash
# Orchestrates a full migration: MySQL dump+restore, then MinIO/S3 mirror.
#
# Usage:
#   cp scripts/migrate.env.example scripts/migrate.env   # fill in real values
#   source scripts/migrate.env
#   ./scripts/migrate_all.sh
#
# Pass --yes to skip the confirmation prompt (e.g. for CI/non-interactive use).
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")"

SKIP_CONFIRM=false
[[ "${1:-}" == "--yes" ]] && SKIP_CONFIRM=true

echo "This will migrate data FROM:"
echo "  MySQL: ${SRC_MYSQL_USER:-?}@${SRC_MYSQL_HOST:-?}:${SRC_MYSQL_PORT:-?}/${SRC_MYSQL_DATABASE:-?}"
echo "  MinIO: ${SRC_MINIO_ENDPOINT:-?}"
echo "TO:"
echo "  MySQL: ${DST_MYSQL_USER:-?}@${DST_MYSQL_HOST:-?}:${DST_MYSQL_PORT:-?}/${DST_MYSQL_DATABASE:-?}"
echo "  MinIO: ${DST_MINIO_ENDPOINT:-?}"
echo
echo "The destination database will be created if missing and the dump loaded into it."
echo "Existing objects in destination buckets are not deleted unless you pass --exact to migrate_minio.sh separately."

if [[ "$SKIP_CONFIRM" != "true" ]]; then
  read -r -p "Proceed? [y/N] " reply
  [[ "$reply" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 1; }
fi

./migrate_mysql.sh
./migrate_minio.sh

echo "==> Migration complete. Verify the destination before pointing services at it."
