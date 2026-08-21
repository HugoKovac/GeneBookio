#!/usr/bin/env bash
# Mirrors every bucket used by the pipeline (books, scripts, audio, prompts —
# see internal/primitive/buckets.go) from the source MinIO/S3 endpoint to the
# destination one, using the MinIO Client (`mc`). Works against any
# S3-compatible destination (MinIO, AWS S3, Spaces, B2, ...).
#
# Install mc if missing: brew install minio/stable/mc
#
# Usage:
#   source scripts/migrate.env
#   ./scripts/migrate_minio.sh          # mirror all buckets (adds/updates only)
#   ./scripts/migrate_minio.sh --exact  # also deletes dest objects not present in source
#   ./scripts/migrate_minio.sh --dry-run
set -euo pipefail

command -v mc >/dev/null 2>&1 || {
  echo "mc (MinIO Client) not found. Install it with: brew install minio/stable/mc" >&2
  exit 1
}

for v in SRC_MINIO_ENDPOINT SRC_MINIO_ACCESS_KEY SRC_MINIO_SECRET_KEY \
         DST_MINIO_ENDPOINT DST_MINIO_ACCESS_KEY DST_MINIO_SECRET_KEY; do
  [[ -n "${!v:-}" ]] || { echo "Missing required env var: $v" >&2; exit 1; }
done

SRC_MINIO_USE_SSL="${SRC_MINIO_USE_SSL:-false}"
DST_MINIO_USE_SSL="${DST_MINIO_USE_SSL:-false}"
BUCKETS=(books scripts audio prompts)

ALIAS_SRC="migrate-src"
ALIAS_DST="migrate-dst"

src_scheme="http"; [[ "$SRC_MINIO_USE_SSL" == "true" ]] && src_scheme="https"
dst_scheme="http"; [[ "$DST_MINIO_USE_SSL" == "true" ]] && dst_scheme="https"

echo "==> Configuring mc aliases"
mc alias set "$ALIAS_SRC" "${src_scheme}://${SRC_MINIO_ENDPOINT}" "$SRC_MINIO_ACCESS_KEY" "$SRC_MINIO_SECRET_KEY" >/dev/null
mc alias set "$ALIAS_DST" "${dst_scheme}://${DST_MINIO_ENDPOINT}" "$DST_MINIO_ACCESS_KEY" "$DST_MINIO_SECRET_KEY" >/dev/null

MIRROR_FLAGS=(--overwrite)
for arg in "$@"; do
  case "$arg" in
    --exact) MIRROR_FLAGS+=(--remove) ;;
    --dry-run) MIRROR_FLAGS+=(--dry-run) ;;
    *) echo "Unknown flag: $arg" >&2; exit 1 ;;
  esac
done

for bucket in "${BUCKETS[@]}"; do
  if ! mc ls "${ALIAS_SRC}/${bucket}" >/dev/null 2>&1; then
    echo "==> Skipping ${bucket}: not found on source"
    continue
  fi
  echo "==> Ensuring destination bucket ${bucket} exists"
  mc mb --ignore-existing "${ALIAS_DST}/${bucket}"
  echo "==> Mirroring ${bucket}: ${ALIAS_SRC} -> ${ALIAS_DST} (${MIRROR_FLAGS[*]})"
  mc mirror "${MIRROR_FLAGS[@]}" "${ALIAS_SRC}/${bucket}" "${ALIAS_DST}/${bucket}"
done

echo "==> Done"
